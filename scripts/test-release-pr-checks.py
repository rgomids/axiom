#!/usr/bin/env python3
"""Regression tests with an isolated GitHub CLI; no remote effects."""
import copy
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

SCRIPT = Path(__file__).resolve().with_name('release-pr-checks.sh')
SHA = 'a' * 40
PR = dict(number=146, state='open', user={'login': 'github-actions[bot]'},
          base={'ref': 'main', 'repo': {'full_name': 'rgomids/axiom'}},
          head={'ref': 'release-please--branches--main', 'sha': SHA,
                'repo': {'full_name': 'rgomids/axiom'}},
          labels=[{'name': 'autorelease: pending'}], title='chore(main): release 0.4.0')
VERSION = '0.4.0'


class ReleaseChecks(unittest.TestCase):
    def run_case(self, prs, ref_sha=SHA, current=None, api_error=False, runs=1, dispatch_error=False, drift_after_ci=False,
                 args=('dispatch', 'rgomids/axiom', VERSION)):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'fixture').write_text(json.dumps(dict(prs=prs, ref_sha=ref_sha,
                current=current if current is not None else (prs[0] if prs else {}),
                api_error=api_error, dispatch_error=dispatch_error, drift_after_ci=drift_after_ci)))
            (root / 'gh').write_text('''#!/usr/bin/env python3
import json, os, sys
from pathlib import Path
r=Path(os.environ['FIXTURE_ROOT']); f=json.loads((r/'fixture').read_text())
a=sys.argv[1:]
if a[0]=='workflow':
    if f['dispatch_error']: sys.exit(1)
    with (r/'effects').open('a') as out: out.write(json.dumps(a)+'\\n')
    sys.exit(0)
if f['api_error']: sys.exit(1)
p=a[-1]
if '?' in p: value=[f['prs']]
elif '/git/ref/' in p: value={'ref':'refs/heads/release-please--branches--main','object':{'type':'commit','sha':f['ref_sha']}}
else:
    value=f['current']
    if f['drift_after_ci'] and (r/'effects').exists(): value['head']['sha']='b'*40
print(json.dumps(value))
''')
            (root / 'gh').chmod(0o755)
            results = [subprocess.run([str(SCRIPT), *args],
                env={**os.environ, 'PATH': f'{root}:{os.environ["PATH"]}',
                     'FIXTURE_ROOT': str(root)}, capture_output=True, text=True)
                for _ in range(runs)]
            effects = (root / 'effects').read_text().splitlines() if (root / 'effects').exists() else []
            return results, effects

    def test_created_updated_unchanged(self):
        # All three action outcomes resolve identical API truth, independent
        # of missing/empty prs_created and pr action outputs.
        for outcome in ('created', 'updated', 'remained the same'):
            with self.subTest(outcome=outcome):
                results, effects = self.run_case([PR])
                self.assertEqual(results[0].returncode, 0, results[0].stderr)
                self.assertEqual([json.loads(e)[2] for e in effects], ['ci.yml', 'delivery-metadata.yml'])

    def test_none_explicit(self):
        results, effects = self.run_case([], args=('resolve', 'rgomids/axiom'))
        self.assertEqual(results[0].returncode, 0)
        self.assertIn('no_open_release_pr', results[0].stdout)
        self.assertEqual(effects, [])

    def test_planned_dispatch_requires_a_release_pr(self):
        # A dispatch from release.sh start must have produced the Release PR.
        self.assert_closed([])

    def test_planned_version_mismatch_gets_no_checks(self):
        # A Release PR whose version differs from the validated plan gets no
        # required checks, so it cannot be merged.
        for version in ('0.3.1', '1.0.0'):
            with self.subTest(version=version):
                self.assert_closed([PR], args=('dispatch', 'rgomids/axiom', version))

    def test_dispatch_requires_planned_version(self):
        for args in (('dispatch', 'rgomids/axiom'), ('dispatch', 'rgomids/axiom', 'v0.4.0'),
                     ('dispatch', 'rgomids/axiom', '0.4.0; true')):
            with self.subTest(args=args):
                self.assert_closed([PR], args=args)

    def test_ambiguous(self):
        self.assert_closed([PR, PR])

    def assert_closed(self, prs, **kwargs):
        results, effects = self.run_case(prs, **kwargs)
        self.assertNotEqual(results[0].returncode, 0)
        self.assertEqual(effects, [])

    def test_invalid_identity(self):
        for path, value in [('user.login', 'rgomids'), ('base.ref', 'other'),
                            ('head.ref', 'other'), ('head.sha', ''),
                            ('head.sha', None), ('head.repo.full_name', 'other/axiom'),
                            ('base.repo.full_name', 'other/axiom'), ('state', 'closed'),
                            ('labels', [])]:
            with self.subTest(path=path):
                pr = copy.deepcopy(PR)
                target = pr
                keys = path.split('.')
                for key in keys[:-1]: target = target[key]
                target[keys[-1]] = value
                self.assert_closed([pr])

    def test_missing_inconsistent_head(self):
        self.assert_closed([PR], ref_sha='b' * 40)
        current = copy.deepcopy(PR)
        current['head']['sha'] = 'b' * 40
        self.assert_closed([PR], current=current)

    def test_resolution_error(self):
        self.assert_closed([PR], api_error=True)

    def test_dispatch_failure(self):
        self.assert_closed([PR], dispatch_error=True)

    def test_drift_after_ci(self):
        results, effects = self.run_case([PR], drift_after_ci=True)
        self.assertNotEqual(results[0].returncode, 0)
        self.assertEqual(len(effects), 1)
        self.assertEqual(json.loads(effects[0])[2], 'ci.yml')

    def test_rerun(self):
        results, effects = self.run_case([PR], runs=2)
        self.assertTrue(all(r.returncode == 0 for r in results))
        self.assertEqual(effects[:2], effects[2:])
        self.assertEqual(len(effects), 4)

    def test_workflow_unconditional_resolution(self):
        workflow = SCRIPT.parent.parent / '.github/workflows/release-please.yml'
        text = workflow.read_text()
        self.assertNotIn('prs_created', text)
        self.assertIn('./scripts/release-pr-checks.sh dispatch "$GITHUB_REPOSITORY" "$PLANNED_VERSION"', text)


if __name__ == '__main__':
    unittest.main()
