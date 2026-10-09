#!/usr/bin/env python3
"""Reconcile only exact monitoring-owned incidents from validated safe evidence."""
import argparse
from datetime import datetime, timezone
import importlib.util
import json
import os
from pathlib import Path
import re
import sys
import urllib.error
import urllib.request

sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location('periodic_monitor', Path(__file__).with_name('periodic-monitor.py'))
monitor = importlib.util.module_from_spec(spec)
spec.loader.exec_module(monitor)
MARKER = '<!-- axiom-periodic:v1:'
BOT = 'github-actions[bot]'
LABELS = ['type:task', 'area:ci-cd']


class SafeError(Exception):
    """Only enum diagnostics may leave the reporting boundary."""


def scopes(layer, scenario):
    if scenario != 'live':
        return {('tidy', 'go', ''), ('integrity', 'go', ''), ('security', 'govulncheck', '')} if layer == 'daily' else {('cli', 'codex', 'exec')}
    if layer == 'daily':
        return {('tidy', 'go', ''), ('integrity', 'go', ''), ('security', 'govulncheck', '')}
    return {('cli', tool, surface['id']) for tool, baseline in monitor.BASELINE['tools'].items()
            for surface in baseline['surfaces']} | {('cli', tool, 'version') for tool in monitor.BASELINE['tools']} | {
                ('simulation', 'go', suite) for suite in monitor.SUITES}


def validate(data, repository, run_id=None):
    """Reject, rather than redact, every unrecognized public field/value."""
    fields = {'schema_version', 'repository', 'workflow', 'run_id', 'run_attempt',
              'run_url', 'timestamp', 'branch', 'subject_sha', 'layer', 'scenario',
              'baseline_digest', 'status', 'checks'}
    if not isinstance(data, dict) or set(data) != fields or data['schema_version'] != 1:
        raise SafeError('invalid_evidence')
    if data['repository'] != repository or data['workflow'] != 'periodic-monitoring':
        raise SafeError('invalid_subject')
    branch = 'main' if repository == 'rgomids/axiom' else os.environ.get('GITHUB_REF_NAME', 'main')
    if data['branch'] != branch or not re.fullmatch(r'[a-z0-9][a-z0-9/-]{0,100}', data['branch']):
        raise SafeError('invalid_subject')
    if not re.fullmatch(r'[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+', repository):
        raise SafeError('invalid_subject')
    if not all(isinstance(data[k], str) and re.fullmatch('[0-9]{1,20}', data[k]) for k in ('run_id', 'run_attempt')):
        raise SafeError('invalid_run')
    if run_id is not None and data['run_id'] != run_id:
        raise SafeError('invalid_run')
    if data['run_url'] != f'https://github.com/{repository}/actions/runs/{data["run_id"]}':
        raise SafeError('invalid_run')
    if not re.fullmatch('[0-9a-f]{40}', data['subject_sha']) or data['baseline_digest'] != monitor.BASELINE_DIGEST:
        raise SafeError('invalid_subject')
    try:
        age = (datetime.now(timezone.utc) - datetime.fromisoformat(data['timestamp'])).total_seconds()
    except (TypeError, ValueError):
        raise SafeError('invalid_timestamp') from None
    if not -300 <= age <= 86400:
        raise SafeError('stale_evidence')
    if data['layer'] not in ('daily', 'weekly') or data['scenario'] not in ('live', 'clean', 'drift', 'infrastructure', 'security'):
        raise SafeError('invalid_layer')
    if repository == 'rgomids/axiom' and data['scenario'] != 'live':
        raise SafeError('sandbox_only')
    if not isinstance(data['checks'], list) or not 1 <= len(data['checks']) <= 32:
        raise SafeError('invalid_checks')
    allowed = scopes(data['layer'], data['scenario'])
    seen = set()
    for row in data['checks']:
        if not isinstance(row, dict):
            raise SafeError('invalid_checks')
        scope = (row.get('check'), row.get('component'), row.get('contract'))
        if scope not in allowed or scope in seen or row.get('category') not in monitor.CATEGORIES:
            raise SafeError('invalid_checks')
        seen.add(scope)
        check, component, contract = scope
        version = row.get('tool_version')
        if not isinstance(version, str) or not re.fullmatch(r'(?:unavailable|(?:go|v)?\d{1,3}\.\d{1,3}(?:\.\d{1,3})?)', version):
            raise SafeError('invalid_version')
        duration = row.get('duration_seconds')
        if type(duration) not in (int, float) or not 0 <= duration <= 300:
            raise SafeError('invalid_duration')
        expected, observed = row.get('baseline'), row.get('observed')
        if check == 'cli' and contract != 'version' and expected is not None:
            baseline = next(s for s in monitor.BASELINE['tools'][component]['surfaces'] if s['id'] == contract)
            if not isinstance(expected, dict) or not isinstance(observed, dict) or not set(expected) <= set(baseline['capabilities']) or set(observed) != set(expected):
                raise SafeError('invalid_snapshot')
            if not all(type(v) is bool for v in list(expected.values()) + list(observed.values())):
                raise SafeError('invalid_snapshot')
            if not all(expected.values()) or (data['scenario'] == 'live' and set(expected) != set(baseline['capabilities'])):
                raise SafeError('invalid_snapshot')
            if row['category'] == 'none' and not all(observed.values()):
                raise SafeError('invalid_snapshot')
        elif check == 'cli' and contract == 'version' and expected is not None:
            if expected != monitor.BASELINE['tools'][component]['version'] or observed != version:
                raise SafeError('invalid_snapshot')
        elif expected is not None or observed is not None:
            raise SafeError('invalid_snapshot')
        reproduction = row.get('reproduction')
        permitted = {monitor.result(check, component)['reproduction']}
        if check in ('tidy', 'integrity'):
            permitted.add('go mod tidy -diff' if check == 'tidy' else 'go mod verify')
        if check == 'security':
            permitted.add('govulncheck ./... (local only; follow SECURITY.md)')
        if check == 'simulation':
            permitted.add(' '.join(['go', 'test', '-json', '-count=1', '-timeout=180s'] + monitor.SUITES[contract]))
        if reproduction not in permitted:
            raise SafeError('invalid_reproduction')
        safe = monitor.result(check, component, row['category'], contract=contract,
                              version=version, duration=duration, expected=expected,
                              observed=observed, reproduction=reproduction)
        if 'tests' in row:
            tests = row['tests']
            if check != 'simulation' or not isinstance(tests, list) or len(tests) > 500:
                raise SafeError('invalid_tests')
            names = monitor.test_names()
            for test in tests:
                if not isinstance(test, dict) or set(test) != {'test', 'status'} or test['test'] not in names or test['status'] not in ('PASS', 'FAIL', 'SKIP'):
                    raise SafeError('invalid_tests')
            safe['tests'] = tests
        if safe != row:
            raise SafeError('invalid_checks')
    status = 'FAIL' if any(c['status'] == 'FAIL' for c in data['checks']) else (
        'INCONCLUSIVE' if any(c['status'] == 'INCONCLUSIVE' for c in data['checks']) else 'PASS')
    if data['status'] != status:
        raise SafeError('invalid_status')
    return data


def sequence(data):
    return [int(data['run_id']), int(data['run_attempt'])]


def incident_body(state):
    # Whole-body ownership: human edits opt out of automated mutation/recovery.
    header = MARKER + json.dumps(state, sort_keys=True, separators=(',', ':')) + ' -->\n\n'
    return header + (f'Periodic monitoring ({state["namespace"]}): **{state["outcome"]}**.\n\n'
                     f'Check: `{state["check"]}`; component: `{state["component"]}`; contract: `{state["contract"]}`; '
                     f'category: `{state["category"]}`.\n\n'
                     f'Occurrences: {state["occurrences"]}; recovery confirmations: {state["passes"]}/2.\n\n'
                     f'Subject: `{state["sha"]}`. [Latest sanitized evidence]({state["url"]}).\n\n'
                     'See docs/development/periodic-monitoring.md for reproduction, categories and manual recovery.\n'
                     'Infrastructure outcomes are inconclusive, not proven product regressions.\n')


def owned(issue, namespace, repository):
    body = issue.get('body') or ''
    if issue.get('user', {}).get('login') != BOT or not body.startswith(MARKER) or 'pull_request' in issue:
        return None
    try:
        state = json.loads(body[len(MARKER):body.index(' -->\n')])
        if set(state) != {'namespace', 'signature', 'layer', 'check', 'component', 'contract', 'category',
                          'baseline', 'occurrences', 'passes', 'last', 'last_failure', 'sha', 'url', 'outcome'}:
            return None
        if state['namespace'] != namespace or state['category'] == 'vulnerability' or state['check'] == 'security':
            return None
        if state['layer'] not in ('daily', 'weekly'):
            return None
        if (state['check'], state['component'], state['contract']) not in scopes(state['layer'], 'live'):
            return None
        if state['category'] not in monitor.CATEGORIES - {'none'} or state['signature'] != monitor.fingerprint(state['check'], state['component'], state['category'], state['contract']):
            return None
        if not re.fullmatch('[0-9a-f]{64}', state['baseline']) or not re.fullmatch('[0-9a-f]{40}', state['sha']):
            return None
        if not all(type(state[k]) is int and 0 <= state[k] <= 1000000 for k in ('occurrences', 'passes')):
            return None
        if state['passes'] > 2 or state['occurrences'] < 1:
            return None
        if not all(isinstance(state[k], list) and len(state[k]) == 2 and all(type(n) is int and n >= 0 for n in state[k]) for k in ('last', 'last_failure')):
            return None
        if state['last_failure'] > state['last']:
            return None
        if state['url'] != f'https://github.com/{repository}/actions/runs/{state["last"][0]}':
            return None
        if state['outcome'] not in ('FAIL', 'INCONCLUSIVE', 'RECOVERING', 'RECOVERED') or body != incident_body(state):
            return None
        return state
    except (ValueError, TypeError, KeyError):
        return None


def plan(data, issues, namespace):
    """Pure reconciliation plan; no recovery on partial, stale or transient coverage."""
    operations = []
    current = sequence(data)
    rows = {(r['check'], r['component'], r['contract']): r for r in data['checks']}
    complete = set(rows) == scopes(data['layer'], data['scenario']) and data['status'] == 'PASS'
    existing = {}
    for issue in issues:
        state = owned(issue, namespace, data['repository'])
        if state:
            if state['signature'] in existing:
                raise SafeError('duplicate_incident_manual_recovery')
            existing[state['signature']] = (issue, state)
    for signature, (issue, old) in existing.items():
        if old['layer'] != data['layer'] or current <= old['last']:
            continue
        scope = (old['check'], old['component'], old['contract'])
        row = rows.get(scope)
        state = dict(old)
        if row and row['fingerprint'] == signature and row['status'] != 'PASS':
            state.update(occurrences=old['occurrences'] + 1, passes=0, last_failure=current,
                         outcome=row['status'], baseline=data['baseline_digest'])
            target = 'open'
        elif complete and row and row['status'] == 'PASS' and old['baseline'] == data['baseline_digest']:
            # A rerun of the same workflow is not an independent confirmation.
            if current[0] == old['last'][0] or issue['state'] == 'closed':
                continue
            state['passes'] += 1
            state['outcome'] = 'RECOVERED' if state['passes'] >= 2 else 'RECOVERING'
            target = 'closed' if state['passes'] >= 2 else 'open'
        elif issue['state'] == 'open' and old['passes']:
            state.update(passes=0, outcome=old['category'] in monitor.INFRA and 'INCONCLUSIVE' or 'FAIL')
            target = 'open'
        else:
            continue
        state.update(last=current, sha=data['subject_sha'], url=data['run_url'])
        operations.append({'op': 'update', 'number': issue['number'], 'state': target,
                           'body': incident_body(state), 'signature': signature})
    for row in data['checks']:
        if row['status'] == 'PASS' or row['check'] == 'security' or row['category'] == 'vulnerability' or row['fingerprint'] in existing:
            continue
        state = {'namespace': namespace, 'signature': row['fingerprint'], 'layer': data['layer'],
                 'check': row['check'], 'component': row['component'], 'contract': row['contract'],
                 'category': row['category'], 'baseline': data['baseline_digest'], 'occurrences': 1,
                 'passes': 0, 'last': current, 'last_failure': current, 'sha': data['subject_sha'],
                 'url': data['run_url'], 'outcome': row['status']}
        operations.append({'op': 'create', 'signature': row['fingerprint'],
                           'title': f'Periodic [{namespace}]: {row["component"]}/{row["contract"] or row["check"]} {row["category"]}',
                           'body': incident_body(state), 'labels': LABELS})
    return operations


class GitHub:
    def __init__(self, repository, token):
        self.base = f'https://api.github.com/repos/{repository}/'
        self.token = token

    def request(self, method, path, payload=None):
        request = urllib.request.Request(self.base + path, method=method,
                    data=json.dumps(payload).encode() if payload is not None else None,
                    headers={'Authorization': 'Bearer ' + self.token, 'Accept': 'application/vnd.github+json',
                             'X-GitHub-Api-Version': '2022-11-28', 'Content-Type': 'application/json'})
        try:
            with urllib.request.urlopen(request, timeout=20) as response:
                raw = response.read(2 * 1024 * 1024 + 1)
            if len(raw) > 2 * 1024 * 1024:
                raise SafeError('output_limit')
            return json.loads(raw)
        except urllib.error.HTTPError as error:
            category = 'permission' if error.code in (401, 403) else ('issues_disabled' if error.code == 410 else 'reporting_api')
            raise SafeError(category) from None
        except (urllib.error.URLError, TimeoutError, ValueError):
            raise SafeError('network') from None

    def issues(self):
        issues = []
        # Server-side creator filter, all states (recurrence must discover closed).
        for page in range(1, 6):
            batch = self.request('GET', f'issues?state=all&creator=github-actions%5Bbot%5D&per_page=100&page={page}')
            if not isinstance(batch, list):
                raise SafeError('invalid_response')
            issues.extend(batch)
            if len(batch) < 100:
                return issues
        raise SafeError('incident_inventory_limit')

    def comments(self, number):
        comments = []
        for page in range(1, 6):
            batch = self.request('GET', f'issues/{number}/comments?per_page=100&page={page}')
            if not isinstance(batch, list):
                raise SafeError('invalid_response')
            comments.extend(batch)
            if len(batch) < 100:
                return comments
        raise SafeError('incident_history_limit')


def reconcile(api, data, namespace):
    operations = plan(data, api.issues(), namespace)
    for operation in operations:
        if operation['op'] == 'create':
            # Re-discover before a write. Workflow-global non-cancelling
            # concurrency is the cross-run lock; discovery handles retries.
            fresh = plan(data, api.issues(), namespace)
            if not any(o['op'] == 'create' and o['signature'] == operation['signature'] for o in fresh):
                continue
            api.request('POST', 'issues', {k: operation[k] for k in ('title', 'body', 'labels')})
        else:
            # Recheck ownership after planning; a human edit opts out.
            issue = api.request('GET', f'issues/{operation["number"]}')
            if not owned(issue, namespace, data['repository']):
                raise SafeError('ownership_changed')
            fresh = plan(data, [issue], namespace)
            match = next((o for o in fresh if o['signature'] == operation['signature']), None)
            if match:
                api.request('PATCH', f'issues/{operation["number"]}',
                            {k: match[k] for k in ('body', 'state')} | {'state_reason': 'completed' if match['state'] == 'closed' else None})
    # Repair the audit comment even when an earlier PATCH/POST succeeded but
    # its response/comment was lost. Same-run retries remain idempotent.
    for issue in api.issues():
        state = owned(issue, namespace, data['repository'])
        if not state or state['last'] != sequence(data):
            continue
        marker = f'<!-- axiom-periodic-event:{namespace}:{data["run_id"]}:{data["run_attempt"]} -->'
        if not any(c.get('user', {}).get('login') == BOT and (c.get('body') or '').startswith(marker + '\n')
                   for c in api.comments(issue['number'])):
            body = (marker + f'\n{state["outcome"]}; occurrences={state["occurrences"]}; '
                    f'recovery={state["passes"]}/2; revision=`{state["sha"]}`. '
                    f'[Sanitized evidence]({state["url"]}).\n')
            api.request('POST', f'issues/{issue["number"]}/comments', {'body': body})
    return len(operations)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--manifest', type=Path, required=True)
    parser.add_argument('--namespace', choices=['production', 'sandbox-255'], default='production')
    args = parser.parse_args()
    repository = os.environ.get('GITHUB_REPOSITORY', '')
    if args.namespace == 'sandbox-255' and repository == 'rgomids/axiom':
        raise SafeError('sandbox_only')
    if args.manifest.stat().st_size > monitor.ARTIFACT_LIMIT:
        raise SafeError('output_limit')
    data = validate(json.loads(args.manifest.read_text(encoding='utf-8')), repository, os.environ.get('GITHUB_RUN_ID'))
    if monitor.execute(['git', 'rev-parse', 'HEAD'])['out'].strip() != data['subject_sha']:
        raise SafeError('invalid_subject')
    if (data['scenario'] != 'live' or repository != 'rgomids/axiom') and args.namespace != 'sandbox-255':
        raise SafeError('sandbox_only')
    token = os.environ.get('GH_TOKEN')
    if not token:
        raise SafeError('permission')
    count = reconcile(GitHub(repository, token), data, args.namespace)
    print(f'Periodic incidents: reconciled {count} owned records; security excluded')


if __name__ == '__main__':
    try:
        main()
    except SafeError as error:
        print(f'Periodic incidents: INCONCLUSIVE ({error})', file=sys.stderr)
        sys.exit(1)
    except (OSError, ValueError, TypeError, KeyError):
        print('Periodic incidents: INCONCLUSIVE (invalid_evidence)', file=sys.stderr)
        sys.exit(1)
