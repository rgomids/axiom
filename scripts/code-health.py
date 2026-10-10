#!/usr/bin/env python3
"""Report-only coverage/complexity snapshots and bounded Actions baseline reuse."""
import argparse
import datetime as dt
import io
import json
import os
from pathlib import Path
import re
import subprocess
import urllib.parse
import urllib.request
import urllib.error
import zipfile

SCHEMA = 1
SCOPE = 'linux/go-test-atomic/default-package-instrumentation;ast/cmd+internal/all-platforms/no-tests/v1'
LIMIT = 15
CRITICAL = {'project', 'projectapp', 'executiongraph', 'graphapplication', 'workflow',
            'runtimeadapter', 'runtimeapplication', 'local', 'portableconfig', 'windowsfs', 'darwinacl'}
MAX_BYTES = 8 * 1024 * 1024
MAX_AGE = dt.timedelta(days=30)
SHA = re.compile(r'^[0-9a-f]{40}$')


def run(*args):
    return subprocess.check_output(args, text=True, encoding='utf-8').strip()


def now():
    return dt.datetime.now(dt.timezone.utc)


def timestamp(value):
    parsed = dt.datetime.fromisoformat(value.replace('Z', '+00:00'))
    if parsed.tzinfo is None:
        raise ValueError('timestamp lacks timezone')
    return parsed


def coverage(text, packages, normalized=None):
    lines = text.splitlines()
    if not lines or lines[0] != 'mode: atomic':
        raise ValueError('missing atomic coverage profile')
    blocks = {}
    for line in lines[1:]:
        match = re.fullmatch(r'(.+):(\d+)\.(\d+),(\d+)\.(\d+) (\d+) (\d+)', line)
        if not match:
            raise ValueError('invalid coverage block')
        file, *numbers = match.groups()
        numbers = list(map(int, numbers))
        if '/' not in file or numbers[0] < 1 or numbers[2] < numbers[0]:
            raise ValueError('invalid coverage location')
        package = file.rsplit('/', 1)[0]
        if package not in packages:
            raise ValueError('coverage package outside measured scope')
        key = (file, *numbers[:4])
        statements, count = numbers[4:]
        if key in blocks and blocks[key][0] != statements:
            raise ValueError('inconsistent duplicate block')
        blocks[key] = (statements, count + blocks.get(key, (0, 0))[1])
    rows = {p: {'statements': 0, 'covered': 0} for p in packages}
    for key, (statements, count) in blocks.items():
        row = rows[key[0].rsplit('/', 1)[0]]
        row['statements'] += statements
        row['covered'] += statements if count else 0
    if normalized is not None:
        normalized.append('mode: atomic')
        for (file, sl, sc, el, ec), (statements, count) in sorted(blocks.items()):
            normalized.append(f'{file}:{sl}.{sc},{el}.{ec} {statements} {count}')
    return rows


def _validate(snapshot, expected_sha=None, fresh=False):
    if snapshot.get('schema_version') != SCHEMA or snapshot.get('scope') != SCOPE:
        raise ValueError('incompatible schema or measured scope')
    if not SHA.fullmatch(snapshot.get('sha', '')) or not SHA.fullmatch(snapshot.get('tree', '')):
        raise ValueError('invalid revision identity')
    if expected_sha and snapshot['sha'] != expected_sha:
        raise ValueError('snapshot SHA differs from trusted run')
    age = now() - timestamp(snapshot['collected_at'])
    if age < -dt.timedelta(minutes=5) or (fresh and age > MAX_AGE):
        raise ValueError('stale or future snapshot')
    if not isinstance(snapshot.get('tools', {}).get('go'), str):
        raise ValueError('missing tool version')
    packages = snapshot['packages']
    if not isinstance(packages, dict) or not packages:
        raise ValueError('missing package inventory')
    for name, row in packages.items():
        if not isinstance(name, str) or not (name == 'github.com/rgomids/axiom' or name.startswith('github.com/rgomids/axiom/')):
            raise ValueError('invalid package identity')
        if any(type(row.get(k)) is not int for k in ('covered', 'statements')):
            raise ValueError('invalid statement count')
        if not 0 <= row['covered'] <= row['statements']:
            raise ValueError('invalid coverage count')
    ids = set()
    for fn in snapshot['functions']:
        if fn['id'] in ids or not fn['id'].startswith(fn['file'] + ':'):
            raise ValueError('duplicate or invalid function identity')
        ids.add(fn['id'])
        if (not fn['file'].startswith(('internal/', 'cmd/')) or '..' in fn['file'].split('/')
                or not fn['file'].endswith('.go') or fn['file'].endswith('_test.go')):
            raise ValueError('invalid function path')
        if type(fn['complexity']) is not int or fn['complexity'] < 1 or type(fn['line']) is not int or fn['line'] < 1:
            raise ValueError('invalid complexity value')
    return snapshot


def validate(snapshot, expected_sha=None, fresh=False):
    try:
        return _validate(snapshot, expected_sha, fresh)
    except (KeyError, TypeError, AttributeError, OverflowError) as error:
        raise ValueError('malformed snapshot schema') from error


def percent(row):
    return 100 * row['covered'] / row['statements'] if row['statements'] else None


def total(packages):
    return {k: sum(row[k] for row in packages.values()) for k in ('covered', 'statements')}


def compare(current, baseline, changed):
    old = {f['id']: f for f in baseline['functions']} if baseline else {}
    entries = []
    for fn in current['functions']:
        if baseline and fn['file'] not in changed:
            continue
        previous = old.get(fn['id'])
        delta = fn['complexity'] - previous['complexity'] if previous else None
        if baseline and previous and delta <= 0:
            continue
        entries.append(dict(fn, baseline=previous['complexity'] if previous else None,
                            delta=delta, status='increase' if previous else 'unmatched/current hotspot'))
    entries.sort(key=lambda x: (-(x['delta'] or 0), -x['complexity'], x['id']))
    return entries[:LIMIT]


def clean(text):
    return re.sub(r'[\x00-\x1f\x7f]', '', str(text)).replace('|', '\\|').replace('`', "'").replace('<', '&lt;').replace('>', '&gt;')[:250]


def display(value):
    return 'unavailable (no statements)' if value is None else f'{value:.2f}%'


def report(current, baseline, changed, reason):
    if baseline and baseline['tools'] != current['tools']:
        baseline, reason = None, 'incompatible tool versions'
    summary = {'schema_version': SCHEMA, 'sha': current['sha'], 'baseline_sha': baseline['sha'] if baseline else None,
               'baseline_state': 'available' if baseline else reason, 'total': total(current['packages']),
               'changed_packages': [], 'complexity': compare(current, baseline, changed),
               'line_diff_coverage': 'not supported'}
    lines = ['## Go code health (report only)', f"Tested SHA: `{current['sha']}`; scope: `{SCOPE}`.",
             f"Total statement coverage: **{display(percent(summary['total']))}**."]
    if baseline:
        delta = percent(summary['total'])
        previous = percent(total(baseline['packages']))
        summary['baseline_collected_at'] = baseline['collected_at']
        summary['baseline_source'] = baseline.get('source')
        lines += [f"Baseline: latest available successful main snapshot `{baseline['sha']}`, collected {clean(baseline['collected_at'])}; age {(now() - timestamp(baseline['collected_at'])).days} days; not merge-base.",
                  f"Total delta: {delta - previous:+.2f} percentage points." if delta is not None and previous is not None else 'Total delta: unavailable.']
    else:
        lines += [f'Baseline comparison: unavailable ({clean(reason)}).']
    lines += ['Changed-package coverage uses PR base-to-tested-revision paths; true line diff coverage is not supported.',
              '', '| Package (changed / critical first; max 15) | Coverage | Delta pp |', '|---|---:|---:|']
    def priority(p):
        path = p.removeprefix('github.com/rgomids/axiom').lstrip('/')
        is_changed = any(str(Path(f).parent).replace('\\', '/') == path for f in changed)
        return (not is_changed, not CRITICAL.intersection(path.split('/')), p)
    for package in sorted(current['packages'], key=priority):
        row = current['packages'][package]
        value = percent(row)
        previous = percent(baseline['packages'][package]) if baseline and package in baseline['packages'] else None
        delta = f'{value - previous:+.2f}' if value is not None and previous is not None else 'unavailable'
        path = package.removeprefix('github.com/rgomids/axiom').lstrip('/')
        is_changed = any(str(Path(f).parent).replace('\\', '/') == path for f in changed)
        if is_changed:
            summary['changed_packages'].append(dict(package=package, **row, delta_pp=(value - previous) if value is not None and previous is not None else None))
        if len([l for l in lines if l.startswith('| ')]) <= LIMIT:
            lines.append(f'| `{clean(package)}` {"(changed)" if is_changed else ""} | {display(value)} | {delta} |')
    lines += ['', '### Complexity: changed-function increases / unmatched hotspots (max 15)',
              'Identity is file + receiver + function; moves/renames are unmatched, never proven regressions. Deleted functions are omitted.',
              '', '| Function | Current | Baseline | Delta |', '|---|---:|---:|---:|']
    for fn in summary['complexity']:
        lines.append(f"| `{clean(fn['id'])}:{fn['line']}` | {fn['complexity']} | {fn['baseline'] if fn['baseline'] is not None else 'unmatched'} | {fn['delta'] if fn['delta'] is not None else 'unavailable'} |")
    if not summary['complexity']:
        lines.append('No relevant complexity increases or unmatched functions.')
    lines += ['', 'Full package inventory and comparable function inventory: snapshot.json artifact. No metric thresholds are enforced.']
    return summary, '\n'.join(lines) + '\n'


class GitHub:
    def __init__(self):
        self.repo = os.environ['GITHUB_REPOSITORY']
        if not re.fullmatch(r'[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+', self.repo):
            raise ValueError('invalid repository')

    def api(self, path):
        req = urllib.request.Request('https://api.github.com/repos/' + self.repo + '/' + path,
            headers={'Authorization': 'Bearer ' + os.environ['GH_TOKEN'], 'Accept': 'application/vnd.github+json',
                     'X-GitHub-Api-Version': '2022-11-28'})
        # Never forward Authorization to artifact storage redirects.
        class NoRedirect(urllib.request.HTTPRedirectHandler):
            def redirect_request(self, req, fp, code, msg, headers, newurl):
                return None
        try:
            with urllib.request.build_opener(NoRedirect).open(req, timeout=20) as response:
                return json.loads(response.read(MAX_BYTES + 1))
        except urllib.error.HTTPError as error:
            if error.code != 302:
                raise
            location = error.headers['Location']
            if urllib.parse.urlsplit(location).scheme != 'https':
                raise ValueError('non-HTTPS artifact redirect')
            with urllib.request.urlopen(location, timeout=20) as response:
                data = response.read(MAX_BYTES + 1)
            if len(data) > MAX_BYTES:
                raise ValueError('artifact exceeds download limit')
            return data

    def snapshot(self, run_id, name, sha):
        artifacts = self.api(f'actions/runs/{int(run_id)}/artifacts?per_page=100')['artifacts']
        matches = [a for a in artifacts if a['name'] == name and not a['expired']]
        if len(matches) != 1:
            raise ValueError('missing, expired or ambiguous artifact')
        archive = self.api(f"actions/artifacts/{int(matches[0]['id'])}/zip")
        with zipfile.ZipFile(io.BytesIO(archive)) as z:
            # Read one known member; never extract archive-controlled paths.
            members = [i for i in z.infolist() if i.filename == 'snapshot.json']
            if len(members) != 1 or members[0].file_size > MAX_BYTES:
                raise ValueError('invalid snapshot archive')
            value = json.loads(z.read(members[0]))
        return validate(value, sha, fresh=True)

    def baseline(self):
        runs = self.api('actions/workflows/code-health-baseline.yml/runs?branch=main&event=push&status=success&per_page=10')['workflow_runs']
        reason = 'no retained successful main snapshot'
        for item in runs:
            try:
                value = self.snapshot(item['id'], 'code-health-main-' + item['head_sha'], item['head_sha'])
                if value.get('source', {}).get('kind') != 'main-tree-equivalent-pr':
                    raise ValueError('invalid baseline provenance')
                return value
            except (ValueError, KeyError, urllib.error.URLError, zipfile.BadZipFile) as error:
                reason = str(error) if isinstance(error, ValueError) else 'artifact inaccessible or malformed'
                continue
        raise ValueError(reason)

    def promote(self, sha):
        commit = self.api('commits/' + sha)
        prs = self.api('commits/' + sha + '/pulls?per_page=30')
        prs = [p for p in prs if p['merged_at'] and p['base']['ref'] == 'main' and p['merge_commit_sha'] == sha
               and p['base']['repo']['full_name'] == self.repo and p['head']['repo'] and p['head']['repo']['full_name'] == self.repo]
        if len(prs) != 1:
            raise ValueError('no unique merged same-repository PR')
        pr = prs[0]
        head = self.api('commits/' + pr['head']['sha'])
        if head['commit']['tree']['sha'] != commit['commit']['tree']['sha']:
            raise ValueError('merged tree differs from PR head')
        runs = self.api('actions/workflows/ci.yml/runs?head_sha=' + pr['head']['sha'] + '&status=success&per_page=10')['workflow_runs']
        for item in runs:
            if item['event'] not in ('pull_request', 'workflow_dispatch') or item['head_repository']['full_name'] != self.repo:
                continue
            try:
                # PR checkout measures the synthetic merge SHA, not head_sha.
                artifacts = self.api(f"actions/runs/{item['id']}/artifacts?per_page=100")['artifacts']
                candidates = [a for a in artifacts if re.fullmatch(r'code-health-linux-[0-9a-f]{40}', a['name']) and not a['expired']]
                if len(candidates) != 1:
                    continue
                tested = candidates[0]['name'].removeprefix('code-health-linux-')
                value = self.snapshot(item['id'], candidates[0]['name'], tested)
                tested_commit = self.api('commits/' + tested)
                if value['tree'] != commit['commit']['tree']['sha'] or value['tree'] != tested_commit['commit']['tree']['sha']:
                    continue
                value.update(sha=sha, source={'kind': 'main-tree-equivalent-pr', 'tested_sha': tested,
                                            'run_id': item['id'], 'pr': pr['number']}, published_at=now().isoformat())
                return value
            except (ValueError, KeyError, urllib.error.URLError, zipfile.BadZipFile):
                continue
        raise ValueError('no successful CI metrics with equivalent tested tree')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=['collect', 'report', 'promote'])
    parser.add_argument('--out', default='coverage')
    parser.add_argument('--base', default=os.environ.get('PR_BASE_SHA', ''))
    args = parser.parse_args()
    out = Path(args.out)
    out.mkdir(parents=True, exist_ok=True)
    if args.mode == 'collect':
        if run('go', 'env', 'GOOS') != 'linux':
            raise ValueError('this snapshot scope requires Linux Go coverage')
        if run('git', 'status', '--porcelain', '--untracked-files=no'):
            raise ValueError('tracked source differs from tested revision')
        packages = run('go', 'list', './...').splitlines()
        normalized = []
        rows = coverage((out / 'coverage.out').read_text(), packages, normalized)
        (out / 'coverage.out').write_text('\n'.join(normalized) + '\n')
        value = {'schema_version': SCHEMA, 'sha': run('git', 'rev-parse', 'HEAD'),
                 'tree': run('git', 'rev-parse', 'HEAD^{tree}'), 'scope': SCOPE,
                 'collected_at': now().isoformat(), 'tools': {'go': run('go', 'version'), 'complexity': 'axiom-ast-v1'},
                 'packages': rows,
                 'functions': json.loads(run('go', 'run', './scripts/codehealth')),
                 'source': {'kind': 'test-run', 'run_id': os.environ.get('GITHUB_RUN_ID', 'local')}}
        validate(value)
        (out / 'snapshot.json').write_text(json.dumps(value, indent=2) + '\n')
    elif args.mode == 'promote':
        value = GitHub().promote(os.environ['GITHUB_SHA'])
        (out / 'snapshot.json').write_text(json.dumps(value, indent=2) + '\n')
    else:
        current = validate(json.loads((out / 'snapshot.json').read_text()))
        baseline, reason = None, 'baseline access unavailable'
        try:
            baseline = GitHub().baseline()
        except ValueError as error:
            reason = str(error)
        except (KeyError, OSError, urllib.error.URLError, zipfile.BadZipFile):
            reason = 'artifact inaccessible or malformed (including unavailable token permissions)'
        changed = []
        if args.base:
            if not SHA.fullmatch(args.base):
                raise ValueError('invalid PR base SHA')
            changed = subprocess.check_output(['git', 'diff', '--name-only', '-z', '--no-renames', args.base, 'HEAD', '--', '*.go']).decode().split('\0')
        summary, markdown = report(current, baseline, changed, reason)
        (out / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
        (out / 'summary.md').write_text(markdown)
        if os.environ.get('GITHUB_STEP_SUMMARY'):
            with open(os.environ['GITHUB_STEP_SUMMARY'], 'a') as stream:
                stream.write(markdown)
        print(markdown)


if __name__ == '__main__':
    main()
