#!/usr/bin/env python3
"""Offline contracts for Issue #245 metrics and trusted baseline reuse."""
import copy
import importlib.util
import io
import json
from pathlib import Path
import sys
import unittest
from unittest.mock import patch
import zipfile

ROOT = Path(__file__).resolve().parent.parent
sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location('health', ROOT / 'scripts/code-health.py')
health = importlib.util.module_from_spec(spec)
spec.loader.exec_module(health)
P = 'github.com/rgomids/axiom/internal/p'
S = 'a' * 40
T = 'b' * 40


def snapshot():
    return dict(schema_version=1, scope=health.SCOPE, sha=S, tree=T,
                collected_at=health.now().isoformat(), tools={'go': 'go version go1.26.0 linux/amd64', 'complexity': 'axiom-ast-v1'},
                packages={P: {'statements': 4, 'covered': 2}},
                functions=[dict(id='internal/p/p.go:Run', file='internal/p/p.go', line=2, complexity=3)],
                source={'kind': 'main-tree-equivalent-pr', 'run_id': 42})


def archive(value, name='snapshot.json'):
    out = io.BytesIO()
    with zipfile.ZipFile(out, 'w') as z:
        z.writestr(name, json.dumps(value))
    return out.getvalue()


class Metrics(unittest.TestCase):
    def test_coverage_duplicates_and_uncovered_inventory(self):
        normalized = []
        rows = health.coverage(f'mode: atomic\n{P}/p.go:3.1,4.2 2 0\n{P}/p.go:1.1,2.2 2 0\n{P}/p.go:1.1,2.2 2 3\n', [P, P+'/empty'], normalized)
        self.assertEqual(rows[P], {'statements': 4, 'covered': 2})
        self.assertIsNone(health.percent(rows[P+'/empty']))
        self.assertEqual(health.percent(health.total(rows)), 50)
        self.assertEqual(normalized, ['mode: atomic', f'{P}/p.go:1.1,2.2 2 3', f'{P}/p.go:3.1,4.2 2 0'])

    def test_invalid_profiles(self):
        for text in ('', 'mode: set', 'mode: atomic\nbroken', f'mode: atomic\n{P}/p.go:0.1,2.2 2 0', 'mode: atomic\nforeign/p.go:1.1,2.2 1 1',
                     f'mode: atomic\n{P}/p.go:1.1,2.2 2 0\n{P}/p.go:1.1,2.2 3 0'):
            with self.subTest(text=text), self.assertRaises(ValueError):
                health.coverage(text, [P])

    def test_schema_identity_counts_freshness(self):
        self.assertEqual(health.validate(snapshot(), S, True)['sha'], S)
        for update in ({'schema_version': 2}, {'sha': T}, {'scope': 'other'}, {'collected_at': '2020-01-01T00:00:00Z'},
                       {'collected_at': '2999-01-01T00:00:00Z'}, {'collected_at': '2026-01-01'},
                       {'packages': {P: {'covered': 5, 'statements': 4}}}, {'functions': [None]}, {'tools': None}):
            value = snapshot()
            value.update(update)
            with self.subTest(update=update), self.assertRaises(ValueError):
                health.validate(value, S, True)
        for value in (None, [], {}, {'schema_version': 1}):
            with self.assertRaises(ValueError):
                health.validate(value)

    def test_increase_rename_delete_empty_diff(self):
        base = snapshot()
        current = copy.deepcopy(base)
        current['functions'][0]['complexity'] = 5
        entries = health.compare(current, base, ['internal/p/p.go'])
        self.assertEqual(entries[0]['delta'], 2)
        self.assertEqual(health.compare(current, base, []), [])
        current['functions'][0]['id'] = 'internal/p/p.go:Renamed'
        self.assertIsNone(health.compare(current, base, ['internal/p/p.go'])[0]['delta'])
        current['functions'] = []
        self.assertEqual(health.compare(current, base, ['internal/p/p.go']), [])

    def test_deterministic_top_n(self):
        current = snapshot()
        current['functions'] = [dict(id=f'internal/p/p.go:F{i:02}', file='internal/p/p.go', line=i+1, complexity=i+1) for i in range(30)]
        entries = health.compare(current, None, [])
        self.assertEqual(len(entries), health.LIMIT)
        self.assertEqual(entries[0]['complexity'], 30)
        current['functions'].reverse()
        self.assertEqual(entries, health.compare(current, None, []))

    def test_report_changed_package_and_unavailable(self):
        current, base = snapshot(), snapshot()
        base['packages'][P]['covered'] = 1
        data, markdown = health.report(current, base, ['internal/p/p_test.go'], '')
        self.assertEqual(data['changed_packages'][0]['delta_pp'], 25)
        self.assertIn('not merge-base', markdown)
        self.assertIn('age 0 days', markdown)
        self.assertIn('50.00%', markdown)
        self.assertEqual(data['complexity'], [])
        data, markdown = health.report(current, None, [], 'permission denied')
        self.assertIsNone(data['baseline_sha'])
        self.assertIn('permission denied', markdown)
        base['tools']['go'] = 'different'
        self.assertIn('incompatible tool versions', health.report(current, base, [], '')[1])
        self.assertNotIn('<script>', health.clean('<script>|`\n'))


class Baselines(unittest.TestCase):
    def setUp(self):
        with patch.dict('os.environ', {'GITHUB_REPOSITORY': 'rgomids/axiom'}):
            self.api = health.GitHub()

    def test_artifact_validation_without_extraction(self):
        artifact = {'id': 1, 'name': 'metrics', 'expired': False}
        for name in ('../snapshot.json', 'snapshot.json'):
            with patch.object(self.api, 'api', side_effect=[{'artifacts': [artifact]}, archive(snapshot(), name)]):
                if name == 'snapshot.json':
                    self.assertEqual(self.api.snapshot(4, 'metrics', S)['sha'], S)
                else:
                    with self.assertRaises(ValueError):
                        self.api.snapshot(4, 'metrics', S)
        with patch.object(self.api, 'api', return_value={'artifacts': [dict(artifact, expired=True)]}), self.assertRaises(ValueError):
            self.api.snapshot(4, 'metrics', S)

    def test_missing_stale_and_permission_baseline(self):
        with patch.object(self.api, 'api', return_value={'workflow_runs': []}), self.assertRaises(ValueError):
            self.api.baseline()
        with patch.object(self.api, 'api', side_effect=PermissionError), self.assertRaises(PermissionError):
            self.api.baseline()
        with patch.object(self.api, 'api', return_value={'workflow_runs': [{'id': 1, 'head_sha': S}]}), patch.object(self.api, 'snapshot', side_effect=ValueError), self.assertRaises(ValueError):
            self.api.baseline()

    def promotion_responses(self, tree=T):
        return [{'commit': {'tree': {'sha': T}}}, [{'merged_at': 'date', 'base': {'ref': 'main', 'repo': {'full_name': 'rgomids/axiom'}},
                'merge_commit_sha': S, 'head': {'sha': T, 'repo': {'full_name': 'rgomids/axiom'}}, 'number': 3}],
                {'commit': {'tree': {'sha': tree}}}, {'workflow_runs': [{'id': 42, 'event': 'pull_request', 'head_repository': {'full_name': 'rgomids/axiom'}}]},
                {'artifacts': [{'name': 'code-health-linux-'+S, 'expired': False}]}, {'commit': {'tree': {'sha': T}}}]

    def test_promotion_requires_success_and_tree_identity(self):
        with patch.object(self.api, 'api', side_effect=self.promotion_responses()), patch.object(self.api, 'snapshot', return_value=snapshot()):
            value = self.api.promote(S)
            self.assertEqual(value['source']['kind'], 'main-tree-equivalent-pr')
            self.assertEqual(value['source']['run_id'], 42)
        with patch.object(self.api, 'api', side_effect=self.promotion_responses('c'*40)), self.assertRaises(ValueError):
            self.api.promote(S)
        with patch.object(self.api, 'api', side_effect=[{}, []]), self.assertRaises(ValueError):
            self.api.promote(S)

    def test_promotion_rejects_forks_failed_runs_and_wrong_tested_tree(self):
        responses = self.promotion_responses()
        responses[1][0]['head']['repo']['full_name'] = 'fork/axiom'
        with patch.object(self.api, 'api', side_effect=responses), self.assertRaises(ValueError):
            self.api.promote(S)
        responses = self.promotion_responses()
        responses[3]['workflow_runs'] = []
        with patch.object(self.api, 'api', side_effect=responses), self.assertRaises(ValueError):
            self.api.promote(S)
        responses = self.promotion_responses()
        responses[-1]['commit']['tree']['sha'] = 'c'*40
        with patch.object(self.api, 'api', side_effect=responses), patch.object(self.api, 'snapshot', return_value=snapshot()), self.assertRaises(ValueError):
            self.api.promote(S)

    def test_workflow_keeps_required_tests_and_report_only(self):
        ci = (ROOT / '.github/workflows/ci.yml').read_text()
        self.assertEqual(ci.count('go test -race -covermode=atomic -coverprofile=coverage/coverage.out ./...'), 1)
        self.assertIn('run: go test ./... -timeout 10m', ci)
        self.assertIn('name: go-quality', ci)
        self.assertNotIn('pull_request_target', ci)
        baseline = (ROOT / '.github/workflows/code-health-baseline.yml').read_text()
        self.assertNotIn('go test', baseline)
        self.assertNotIn('write', baseline)


if __name__ == '__main__':
    unittest.main()
