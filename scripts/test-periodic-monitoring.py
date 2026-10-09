#!/usr/bin/env python3
"""Offline acceptance matrix for periodic checks, safe evidence and incident lifecycle."""
import copy
from datetime import datetime, timedelta, timezone
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.dont_write_bytecode = True
ROOT = Path(__file__).resolve().parent.parent


def load(name, path):
    spec = importlib.util.spec_from_file_location(name, ROOT / 'scripts' / path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


monitor = load('monitor', 'periodic-monitor.py')
incidents = load('incidents', 'periodic-incidents.py')
REPO = 'sandbox/axiom'
CANARY = 'PRIVATE_CANARY_255'


def response(out='', category='none', code=0):
    return {'out': out, 'err': CANARY, 'category': category, 'code': code, 'duration': 0.1}


def evidence(run=1, category='none', extra=None):
    with patch.dict(os.environ, {'GITHUB_REPOSITORY': REPO, 'GITHUB_RUN_ID': str(run), 'GITHUB_RUN_ATTEMPT': '1'}):
        rows = [monitor.result('cli', 'codex', category, contract='exec')]
        if extra:
            rows.extend(extra)
        return monitor.manifest('weekly', rows, 'clean' if category == 'none' else 'drift')


def issue(operation, number=1, state='open'):
    return {'number': number, 'state': state, 'body': operation['body'], 'user': {'login': incidents.BOT}}


class Checks(unittest.TestCase):
    def test_live_command_bounded_and_sanitized(self):
        found = monitor.execute([sys.executable, '-c', f'print("{CANARY}")'])
        self.assertEqual(found['category'], 'none')
        row = monitor.result('tidy', 'go', found['category'], duration=found['duration'])
        self.assertNotIn(CANARY, json.dumps(row))

    def test_missing_binary(self):
        found = monitor.execute(['axiom-no-such-cli-255'])
        self.assertEqual(found['category'], 'missing_binary')

    def test_nonzero_and_network_distinct(self):
        for message, expected in [('bad arguments', 'nonzero_exit'), ('network connection failed', 'network')]:
            with self.subTest(message=message):
                found = monitor.execute([sys.executable, '-c', f'import sys; print({message!r},file=sys.stderr); sys.exit(2)'])
                self.assertEqual(found['category'], expected)

    def test_timeout_is_bounded(self):
        found = monitor.execute([sys.executable, '-c', 'import time; time.sleep(30)'], timeout=0.2)
        self.assertEqual(found['category'], 'timeout')
        self.assertLess(found['duration'], 8)

    @unittest.skipIf(os.name == 'nt', 'POSIX runner process-group contract')
    def test_child_pipe_lifetime_is_bounded(self):
        code = 'import subprocess,sys; subprocess.Popen([sys.executable,"-c","import time;time.sleep(30)"])'
        found = monitor.execute([sys.executable, '-c', code], timeout=0.3)
        self.assertEqual(found['category'], 'timeout')
        self.assertLess(found['duration'], 8)

    def test_oversize_stdout_and_stderr(self):
        found = monitor.execute([sys.executable, '-c', 'import sys;print("x"*600000);print("y"*600000,file=sys.stderr)'])
        self.assertEqual(found['category'], 'output_limit')
        self.assertLessEqual(len(found['out']), monitor.OUTPUT_LIMIT)
        self.assertLessEqual(len(found['err']), monitor.OUTPUT_LIMIT)

    def test_clean_environment_omits_credentials_and_user_config(self):
        with patch.dict(os.environ, {'GH_TOKEN': CANARY, 'OPENAI_API_KEY': CANARY, 'ANTHROPIC_API_KEY': CANARY,
                                   'CODEX_HOME': '/private', 'GOFLAGS': CANARY, 'HTTPS_PROXY': CANARY}):
            env = monitor.clean_environment(Path('/isolated'))
        self.assertNotIn(CANARY, json.dumps(env))
        self.assertEqual(env['CODEX_HOME'], str(Path('/isolated/codex')))
        self.assertEqual(env['GOTOOLCHAIN'], 'local')

    def test_semantic_contract_ignores_spacing_dates_and_secrets(self):
        surface = monitor.BASELINE['tools']['codex']['surfaces'][1]
        found = response('Usage: exec 2026-10-09\n--model X\n--sandbox Y\n--json\n' + CANARY)
        row = monitor.check_surface('codex', surface, found, '0.162.1')
        self.assertEqual(row['status'], 'PASS')
        self.assertNotIn(CANARY, json.dumps(row))
        self.assertNotIn('2026-10-09', json.dumps(row))

    def test_removed_renamed_flag_is_drift(self):
        surface = monitor.BASELINE['tools']['codex']['surfaces'][1]
        row = monitor.check_surface('codex', surface, response('Usage: exec --models --sandbox --json'), '0.162.1')
        self.assertEqual(row['category'], 'contract_drift')
        self.assertFalse(row['observed']['--model'])

    def test_empty_malformed_and_nonzero_help(self):
        surface = monitor.BASELINE['tools']['codex']['surfaces'][1]
        for out, category, expected in [('', 'none', 'malformed_output'), ('garbage', 'none', 'malformed_output'),
                                        ('Usage: exec', 'nonzero_exit', 'nonzero_exit'), ('', 'timeout', 'timeout')]:
            with self.subTest(expected=expected):
                row = monitor.check_surface('codex', surface, response(out, category), '0.162.1')
                self.assertEqual(row['category'], expected)

    def test_version_parsers_do_not_retain_tail(self):
        cases = [('gh', 'gh version 2.102.0 (date) ' + CANARY, '2.102.0'),
                 ('codex', 'codex-cli 0.162.1\n' + CANARY, '0.162.1'),
                 ('claude', '2.1.296 (Claude Code)\n' + CANARY, '2.1.296'),
                 ('go', 'go version go1.26.0 linux/amd64', 'go1.26.0')]
        for tool, out, expected in cases:
            self.assertEqual(monitor.parse_version(tool, out), expected)
        self.assertEqual(monitor.parse_version('codex', CANARY), 'unavailable')

    def test_security_pass_fail_malformed_network_privacy(self):
        cases = [(' {"config":{}}\n{"progress":{}}', 'none', 'none'),
                 ('{"config":{}}\n{"finding":{"osv":"' + CANARY + '"}}', 'none', 'vulnerability'),
                 ('{"config":{}}\n{"finding":{}}', 'nonzero_exit', 'vulnerability'),
                 ('{"config":{}}', 'nonzero_exit', 'nonzero_exit'),
                 ('', 'none', 'malformed_output'), ('[]', 'none', 'malformed_output'),
                 (CANARY, 'network', 'network')]
        for out, category, expected in cases:
            with self.subTest(expected=expected):
                got = monitor.security_category(response(out, category))
                self.assertEqual(got, expected)
                self.assertNotIn(CANARY, json.dumps(monitor.result('security', 'govulncheck', got)))

    def test_install_failure_is_not_a_pass(self):
        with patch.object(monitor, 'install', return_value=('/missing', response('', 'network'))), patch.object(monitor, 'execute', return_value=response('')):
            rows = monitor.weekly(Path('/tmp'), {})
        self.assertTrue(all(r['status'] == 'INCONCLUSIVE' for r in rows if r['check'] == 'cli'))

    def test_unsupported_version_refuses_surface_probes(self):
        def command(argv, *args, **kwargs):
            if '--version' in argv:
                return response('codex-cli 0.1.0')
            if 'version' in argv:
                return response('go version go1.26.0 linux/amd64')
            return response('')
        with patch.object(monitor, 'install', return_value=('/codex', response())), patch.object(monitor, 'execute', side_effect=command):
            rows = monitor.weekly(Path('/tmp'), {})
        row = next(r for r in rows if r['component'] == 'codex')
        self.assertEqual(row['category'], 'unsupported_version')
        self.assertEqual(len([r for r in rows if r['component'] == 'codex']), 1)

    def test_dependency_hygiene_is_readonly_and_distinguishes_environment(self):
        for category, expected in [('none', 'PASS'), ('nonzero_exit', 'FAIL'), ('network', 'INCONCLUSIVE')]:
            calls = []
            def command(argv, *args, **kwargs):
                calls.append(argv)
                return response('go version go1.26.0 linux/amd64', category)
            with patch.object(monitor, 'execute', side_effect=command), patch.object(monitor, 'install', return_value=('/missing', response('', 'installation'))):
                rows = monitor.daily(Path('/tmp'), {})
            self.assertEqual(rows[0]['status'], expected)
            self.assertIn(['go', 'mod', 'tidy', '-diff'], calls)
            self.assertNotIn(['go', 'mod', 'tidy'], calls)

    def test_per_test_detail_excludes_dynamic_private_names_and_output(self):
        name = sorted(monitor.test_names())[0]
        raw = '\n'.join(json.dumps({'Action': 'pass', 'Test': test, 'Output': CANARY}) for test in [name, CANARY, name + '/' + CANARY])
        tests = monitor.parse_go_tests(response(raw))
        self.assertEqual(tests, [{'test': name, 'status': 'PASS'}])
        self.assertNotIn(CANARY, json.dumps(tests))

    def test_fingerprint_stability_and_component_classification(self):
        one, two = evidence(1, 'contract_drift'), evidence(2, 'contract_drift')
        self.assertEqual(one['checks'][0]['fingerprint'], two['checks'][0]['fingerprint'])
        self.assertNotEqual(one['checks'][0]['fingerprint'], evidence(3, 'network')['checks'][0]['fingerprint'])


class Evidence(unittest.TestCase):
    def test_schema_structure_matches_manifest_and_check_contract(self):
        schema = json.loads((ROOT / 'scripts/periodic-evidence.schema.json').read_text())
        data = evidence()
        self.assertEqual(set(schema['required']), set(data))
        self.assertEqual(set(schema['properties']), set(data))
        row_schema = schema['properties']['checks']['items']
        self.assertEqual(set(row_schema['required']), set(data['checks'][0]))
        self.assertEqual(set(row_schema['properties']['category']['enum']), monitor.CATEGORIES)
        self.assertFalse(schema['additionalProperties'])
        self.assertFalse(row_schema['additionalProperties'])

    def test_clean_versioned_manifest_and_summary(self):
        data = evidence()
        incidents.validate(data, REPO)
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory)
            monitor.write_evidence(data, output)
            self.assertEqual(json.loads((output / 'manifest.json').read_text()), data)
            self.assertIn('PASS', (output / 'summary.md').read_text())
            self.assertLess((output / 'manifest.json').stat().st_size, monitor.ARTIFACT_LIMIT)

    def test_all_public_fields_reject_sensitive_payloads(self):
        data = evidence()
        for key in data:
            if key in ('checks',):
                continue
            poisoned = copy.deepcopy(data)
            poisoned[key] = CANARY
            with self.subTest(field=key), self.assertRaises((incidents.SafeError, ValueError)):
                incidents.validate(poisoned, REPO)
        for key in data['checks'][0]:
            poisoned = copy.deepcopy(data)
            poisoned['checks'][0][key] = CANARY
            with self.subTest(check_field=key), self.assertRaises((incidents.SafeError, ValueError, TypeError)):
                incidents.validate(poisoned, REPO)

    def test_unknown_fields_and_bad_snapshots_refused(self):
        data = evidence()
        data['private'] = CANARY
        with self.assertRaises(incidents.SafeError):
            incidents.validate(data, REPO)
        data = evidence()
        data['checks'][0]['baseline'] = {'--secret': True}
        data['checks'][0]['observed'] = {'--secret': True}
        with self.assertRaises(incidents.SafeError):
            incidents.validate(data, REPO)

    def test_stale_future_duplicate_and_subject_mismatch(self):
        for mutate in [lambda d: d.update(timestamp=(datetime.now(timezone.utc) - timedelta(days=2)).isoformat()),
                       lambda d: d.update(timestamp=(datetime.now(timezone.utc) + timedelta(days=2)).isoformat()),
                       lambda d: d['checks'].append(d['checks'][0]), lambda d: d.update(subject_sha='main'),
                       lambda d: d.update(branch='feature'), lambda d: d.update(status='FAIL')]:
            data = evidence()
            mutate(data)
            with self.assertRaises(incidents.SafeError):
                incidents.validate(data, REPO)

    def test_security_public_evidence_has_no_advisory(self):
        with patch.dict(os.environ, {'GITHUB_REPOSITORY': REPO}):
            data = monitor.manifest('daily', monitor.fixture('daily', 'security'), 'security')
        incidents.validate(data, REPO)
        self.assertEqual(incidents.plan(data, [], 'sandbox-255'), [])
        with tempfile.TemporaryDirectory() as directory:
            monitor.write_evidence(data, Path(directory))
            self.assertIn('SECURITY.md', (Path(directory) / 'summary.md').read_text())

    def test_artifact_size_limit(self):
        data = evidence()
        data['private'] = CANARY * 100000
        with tempfile.TemporaryDirectory() as directory, self.assertRaises(ValueError):
            monitor.write_evidence(data, Path(directory))

    def test_production_fixture_refused_before_execution(self):
        with tempfile.TemporaryDirectory() as directory:
            env = dict(os.environ, GITHUB_REPOSITORY='rgomids/axiom')
            found = subprocess.run([sys.executable, str(ROOT / 'scripts/periodic-monitor.py'), 'run', '--layer', 'weekly',
                                    '--scenario', 'drift', '--output', directory], capture_output=True, env=env)
            self.assertNotEqual(found.returncode, 0)
            self.assertEqual(list(Path(directory).iterdir()), [])


class Lifecycle(unittest.TestCase):
    def test_api_lifecycle_and_audit_repair_are_idempotent(self):
        class API:
            def __init__(self):
                self.records = []
                self.history = []
                self.fail_comment = False
            def issues(self):
                return self.records
            def comments(self, number):
                return self.history
            def request(self, method, path, payload=None):
                if method == 'GET':
                    return self.records[0]
                if method == 'POST' and path == 'issues':
                    self.records.append(issue(payload))
                    return self.records[0]
                if method == 'PATCH':
                    self.records[0].update(body=payload['body'], state=payload['state'])
                    return self.records[0]
                if self.fail_comment:
                    self.fail_comment = False
                    raise incidents.SafeError('network')
                self.history.append({'body': payload['body'], 'user': {'login': incidents.BOT}})
                return self.history[-1]
        api = API()
        api.fail_comment = True
        with self.assertRaises(incidents.SafeError):
            incidents.reconcile(api, evidence(1, 'contract_drift'), 'sandbox-255')
        self.assertEqual(len(api.records), 1)
        self.assertEqual(len(api.history), 0)
        incidents.reconcile(api, evidence(1, 'contract_drift'), 'sandbox-255')
        incidents.reconcile(api, evidence(1, 'contract_drift'), 'sandbox-255')
        self.assertEqual(len(api.history), 1)
        for run, category in [(2, 'contract_drift'), (3, 'none'), (4, 'none')]:
            incidents.reconcile(api, evidence(run, category), 'sandbox-255')
        self.assertEqual(api.records[0]['state'], 'closed')
        incidents.reconcile(api, evidence(5, 'contract_drift'), 'sandbox-255')
        self.assertEqual(api.records[0]['state'], 'open')
        self.assertEqual(len(api.records), 1)
        self.assertEqual(len(api.history), 5)
        self.assertNotIn(CANARY, json.dumps(api.records + api.history))

    def test_discovery_race_does_not_create_duplicate(self):
        data = evidence(1, 'contract_drift')
        other = issue(incidents.plan(data, [], 'sandbox-255')[0])
        class API:
            def __init__(self):
                self.reads = 0
                self.writes = []
            def issues(self):
                self.reads += 1
                return [] if self.reads == 1 else [other]
            def comments(self, number):
                return []
            def request(self, method, path, payload=None):
                self.writes.append(path)
        api = API()
        incidents.reconcile(api, data, 'sandbox-255')
        self.assertNotIn('issues', api.writes)

    def test_create_dedup_recovery_recurrence(self):
        first = incidents.plan(evidence(1, 'contract_drift'), [], 'sandbox-255')
        self.assertEqual(len(first), 1)
        tracked = issue(first[0])
        self.assertEqual(incidents.plan(evidence(1, 'contract_drift'), [tracked], 'sandbox-255'), [])
        update = incidents.plan(evidence(2, 'contract_drift'), [tracked], 'sandbox-255')[0]
        self.assertEqual(update['op'], 'update')
        tracked['body'] = update['body']
        self.assertEqual(incidents.owned(tracked, 'sandbox-255', REPO)['occurrences'], 2)
        recover = incidents.plan(evidence(3), [tracked], 'sandbox-255')[0]
        self.assertEqual(recover['state'], 'open')
        tracked['body'] = recover['body']
        close = incidents.plan(evidence(4), [tracked], 'sandbox-255')[0]
        self.assertEqual(close['state'], 'closed')
        tracked.update(body=close['body'], state='closed')
        reopen = incidents.plan(evidence(5, 'contract_drift'), [tracked], 'sandbox-255')[0]
        self.assertEqual(reopen['state'], 'open')
        self.assertEqual(incidents.owned(issue(reopen), 'sandbox-255', REPO)['occurrences'], 3)

    def test_stale_out_of_order_reruns_and_partial_coverage_do_not_close(self):
        tracked = issue(incidents.plan(evidence(10, 'contract_drift'), [], 'sandbox-255')[0])
        self.assertEqual(incidents.plan(evidence(9), [tracked], 'sandbox-255'), [])
        rerun = evidence(10)
        rerun['run_attempt'] = '2'
        self.assertEqual(incidents.plan(rerun, [tracked], 'sandbox-255'), [])
        partial = evidence(11)
        partial['scenario'] = 'live'
        self.assertEqual(incidents.plan(partial, [tracked], 'sandbox-255'), [])

    def test_infrastructure_and_baseline_change_cannot_confirm_recovery(self):
        tracked = issue(incidents.plan(evidence(1, 'contract_drift'), [], 'sandbox-255')[0])
        recovery = incidents.plan(evidence(2), [tracked], 'sandbox-255')[0]
        tracked['body'] = recovery['body']
        interrupted = incidents.plan(evidence(3, 'network'), [tracked], 'sandbox-255')
        update = next(o for o in interrupted if o['op'] == 'update')
        self.assertEqual(update['state'], 'open')
        self.assertEqual(incidents.owned(issue(update), 'sandbox-255', REPO)['passes'], 0)
        changed = evidence(4)
        changed['baseline_digest'] = 'b' * 64
        self.assertFalse(any(o['state'] == 'closed' for o in incidents.plan(changed, [tracked], 'sandbox-255')))

    def test_unrelated_human_edited_and_other_namespace_issues_untouched(self):
        owned = issue(incidents.plan(evidence(1, 'contract_drift'), [], 'sandbox-255')[0])
        altered = copy.deepcopy(owned)
        altered['body'] += '\nHuman-managed'
        human = copy.deepcopy(owned)
        human['user']['login'] = 'maintainer'
        self.assertEqual(incidents.plan(evidence(2), [altered, human, {'body': 'unrelated'}], 'sandbox-255'), [])
        self.assertEqual(incidents.plan(evidence(2), [owned], 'production'), [])

    def test_duplicate_inventory_refuses_mutation(self):
        tracked = issue(incidents.plan(evidence(1, 'contract_drift'), [], 'sandbox-255')[0])
        with self.assertRaises(incidents.SafeError):
            incidents.plan(evidence(2), [tracked, dict(tracked, number=2)], 'sandbox-255')

    def test_infrastructure_incident_explicitly_inconclusive(self):
        operation = incidents.plan(evidence(1, 'network'), [], 'sandbox-255')[0]
        self.assertIn('INCONCLUSIVE', operation['body'])
        self.assertIn('network', operation['title'])
        self.assertNotIn(CANARY, operation['body'])

    def test_permission_and_network_are_safe_errors(self):
        import urllib.error
        for error, expected in [(urllib.error.HTTPError('url', 403, CANARY, {}, None), 'permission'),
                                (urllib.error.URLError(CANARY), 'network')]:
            api = incidents.GitHub(REPO, CANARY)
            with patch('urllib.request.urlopen', side_effect=error), self.assertRaises(incidents.SafeError) as raised:
                api.issues()
            self.assertEqual(str(raised.exception), expected)

    def test_idempotent_retry_after_unknown_create_result(self):
        class API:
            def __init__(self):
                self.records = []
                self.creates = 0
            def issues(self):
                return self.records
            def comments(self, number):
                return []
            def request(self, method, path, payload=None):
                if method == 'POST' and path == 'issues':
                    self.creates += 1
                    self.records.append(issue(payload))
                    raise incidents.SafeError('network')
                return self.records[0]
        api = API()
        with self.assertRaises(incidents.SafeError):
            incidents.reconcile(api, evidence(1, 'contract_drift'), 'sandbox-255')
        incidents.reconcile(api, evidence(1, 'contract_drift'), 'sandbox-255')
        self.assertEqual(api.creates, 1)


class Workflow(unittest.TestCase):
    def test_isolation_permissions_pins_timeouts_and_retention(self):
        workflow = (ROOT / '.github/workflows/periodic-monitoring.yml').read_text()
        for forbidden in ('pull_request:', 'push:', 'workflow_run:', 'continue-on-error:', 'secrets.'):
            self.assertNotIn(forbidden, workflow)
        self.assertIn("cron: '17 5 * * *'", workflow)
        self.assertIn("cron: '37 6 * * 1'", workflow)
        self.assertIn('workflow_dispatch:', workflow)
        self.assertIn('cancel-in-progress: false', workflow)
        self.assertIn('retention-days: 14', workflow)
        analysis, reporter = workflow.split('  incidents:', 1)
        self.assertNotIn('issues: write', analysis)
        self.assertIn('issues: write', reporter)
        self.assertIn("if: always() && needs.analysis.outputs.artifact == 'success'", reporter)
        self.assertIn('if-no-files-found: error', workflow)
        import re
        for line in workflow.splitlines():
            if 'uses:' in line:
                self.assertRegex(line, r'@[0-9a-f]{40}(?:\s|$)')
        for job in ('analysis', 'incidents'):
            self.assertIn('timeout-minutes:', workflow.split(f'  {job}:', 1)[1])
        self.assertIn("github.repository == 'rgomids/axiom' && 'main' || github.ref", workflow)
        self.assertIn('ref: ${{ needs.analysis.outputs.subject }}', reporter)
        for path in (ROOT / '.github/workflows').glob('*.yml'):
            if path.name != 'periodic-monitoring.yml':
                self.assertNotIn('periodic-monitor', path.read_text())
        self.assertNotIn('periodic', (ROOT / '.github/rulesets/main.json').read_text())


if __name__ == '__main__':
    unittest.main()
