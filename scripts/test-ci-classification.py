#!/usr/bin/env python3
"""Deterministic path fixtures and fail-closed required-check contracts."""
import importlib.util
import json
import os
import re
from pathlib import Path
import subprocess
import tempfile
import unittest
import sys
from unittest.mock import patch

ROOT = Path(__file__).resolve().parent.parent
sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location('classifier', ROOT / 'scripts/classify-ci-changes.py')
classifier = importlib.util.module_from_spec(spec)
spec.loader.exec_module(classifier)


class Routing(unittest.TestCase):
    def test_fixture_matrix(self):
        fixtures = [
            (['docs/development/guide.md', 'README.md'], []),
            (['site/index.html', 'docs/assets/logo.svg'], []),
            (['.agents/skills/axiom-review/SKILL.md'], []),
            (['internal/application/run.go'], ['go', 'verify', 'upgrade']),
            (['go.mod', 'go.sum'], ['go', 'verify', 'upgrade']),
            (['install.ps1'], ['go', 'verify', 'release', 'upgrade']),
            (['internal/compatibility/testdata/state.json'], ['go', 'verify', 'release', 'upgrade']),
            (['scripts/test-release-flow.sh'], ['release']),
            (['scripts/build-release-archives.sh'], ['go', 'verify', 'release', 'upgrade']),
            (['.github/workflows/ci.yml'], ['go', 'verify', 'release', 'upgrade']),
            (['new unknown file'], ['go', 'verify', 'release', 'upgrade']),
            (['docs/x.md', 'cmd/lingo/main.go'], ['go', 'verify', 'release', 'upgrade']),
            ([], ['go', 'verify', 'release', 'upgrade']),
        ]
        for paths, expected in fixtures:
            with self.subTest(paths=paths):
                result = classifier.classify(paths)
                self.assertEqual([key for key in ('go', 'verify', 'release', 'upgrade') if result[key]], expected)
        self.assertTrue(classifier.classify(['docs/x.md'], True)['full'])

    def test_real_diff_includes_both_rename_sides_and_deleted_paths(self):
        with tempfile.TemporaryDirectory() as directory:
            def git(*args):
                return subprocess.check_output(['git', '-C', directory, *args]).decode().strip()
            git('init', '-q')
            git('config', 'user.email', 'fixture@example.invalid')
            git('config', 'user.name', 'Fixture')
            git('config', 'commit.gpgsign', 'false')
            root = Path(directory)
            (root / 'docs').mkdir()
            (root / 'docs/old.md').write_text('fixture')
            git('add', '.')
            git('commit', '-qm', 'base')
            base = git('rev-parse', 'HEAD')
            (root / 'docs/old.md').rename(root / 'unknown.txt')
            git('add', '-A')
            git('commit', '-qm', 'head')
            output = subprocess.check_output(['python', str(ROOT / 'scripts/classify-ci-changes.py'),
                                              '--base', base, '--head', 'HEAD'], cwd=directory)
            result = json.loads(output)
            self.assertEqual(set(result['paths']), {'docs/old.md', 'unknown.txt'})
            self.assertTrue(result['full'])

    def test_required_contexts_and_failure_propagation(self):
        workflow = (ROOT / '.github/workflows/ci.yml').read_text()
        for job, flag in [('go-quality', 'go'), ('verify', 'verify'),
                          ('release-contract', 'release'), ('upgrade-journeys', 'upgrade')]:
            with self.subTest(job=job):
                block = re.split(r'\n  [a-z][a-z-]*:\n', workflow.split('\n  ' + job + ':\n')[1], 1)[0]
                self.assertIn('if: always()', block)
                self.assertIn('needs: [classify, repository, ' + job + '-suite]', block)
                self.assertIn('--conclude --suite ' + flag, block)
                self.assertIn('ci-plan-${{ github.run_attempt }}', block)
                if job in ('verify', 'upgrade-journeys'):
                    self.assertIn('ci-row-' + flag + '-${{ matrix.platform }}-${{ github.run_attempt }}', block)
                self.assertNotIn('continue-on-error', block)
                for classification, repository, selected, result, passed in [
                    ('success', 'success', 'true', 'success', True),
                    ('success', 'success', 'false', 'skipped', True),
                    ('failure', 'success', 'false', 'skipped', False),
                    ('success', 'failure', 'false', 'skipped', False),
                    ('cancelled', 'success', 'true', 'success', False),
                    ('success', 'success', 'true', 'failure', False),
                    ('success', 'success', 'true', 'cancelled', False),
                    ('success', 'success', 'true', 'skipped', False),
                    ('success', 'success', '', 'skipped', False),
                    ('success', 'success', 'false', 'success', False),
                ]:
                    plan = classifier.plan([dict(path='unknown', status='M')]) if selected == 'true' else classifier.plan([dict(path='docs/x.md', status='M')])
                    self.assertEqual(classifier.conclude(plan, selected, result,
                        classification, repository, suite=flag), passed,
                        (job, classification, repository, selected, result))
                suite = re.split(r'\n  [a-z][a-z-]*:\n', workflow.split('\n  ' + job + '-suite:\n')[1], 1)[0]
                self.assertIn("if: needs.classify.outputs." + flag + " == 'true'", suite)
        self.assertIn('python3 scripts/classify-ci-changes.py --full', workflow)
        self.assertNotIn('  push:', workflow)
        self.assertIn('python3 scripts/test-ci-classification.py', workflow)

    def test_sensitive_paths_force_every_physical_suite(self):
        for path in ('.github/rulesets/main.json', '.agents/policies/security.md',
                     '.agents/policies/quality.md', '.github/workflows/deploy-landpage.yml',
                     '.github/workflows/ci.yml', '.github/CODEOWNERS', 'AGENTS.md',
                     'CLAUDE.md', 'SECURITY.md', 'docs/security/repository-security.md'):
            with self.subTest(path=path):
                result = classifier.plan([dict(path=path, status='M')])
                self.assertEqual(result['reason'], 'sensitive_paths')
                self.assertTrue(all(result[key] for key in classifier.SUITES))

    def test_plan_schema_and_na_reasons(self):
        for entries, full, reason in [
            ([dict(path='docs/x.md', status='D')], False, 'single_category'),
            ([dict(path='docs/x.md', status='M'), dict(path='cmd/main.go', status='A')],
             False, 'mixed_categories'),
            ([], True, 'manual_dispatch'), ([], False, 'empty_diff')]:
            result = classifier.plan(entries, 'base', 'head', full)
            self.assertEqual(result['schema_version'], 1)
            self.assertEqual(result['identity'], dict(classifier.identity(), base='base', head='head'))
            self.assertEqual(result['reason'], reason)
            with patch.dict(os.environ, {'GITHUB_EVENT_NAME': 'workflow_dispatch' if full else 'pull_request'}):
                validated = classifier.plan(entries, 'base', 'head', full)
                self.assertTrue(classifier.valid_plan(validated))
            self.assertEqual([x['status'] for x in result['changes']], [x['status'] for x in entries])
            for suite in classifier.SUITES:
                self.assertEqual(result['suites'][suite]['selected'], result[suite])
                self.assertTrue(result['suites'][suite]['reason'])

    def test_malformed_or_spoofed_na_plan_fails(self):
        plan = classifier.plan([dict(path='cmd/main.go', status='M')])
        for mutated in (dict(plan, schema_version=2), dict(plan, paths=[]),
                        dict(plan, changes=[]), dict(plan, reason='single_category_spoof'),
                        dict(plan, go=False), dict(plan, suites={}), dict(plan, identity={})):
            self.assertFalse(classifier.valid_plan(mutated))
        plan['suites']['go'] = dict(selected=False, reason='not_applicable_to_docs')
        self.assertFalse(classifier.conclude(plan, 'false', 'skipped',
            'success', 'success', suite='go'))

    def test_missing_or_invalid_platform_cannot_pass(self):
        plan = classifier.plan([dict(path='unknown', status='M')])
        for suite, platforms in [('verify', ['linux', 'macos', 'windows']),
                                 ('upgrade', ['linux', 'macos'])]:
            for platform in platforms:
                row = dict(schema_version=1, identity=classifier.identity(), suite=suite,
                           platform=platform, result='success')
                self.assertTrue(classifier.conclude(plan, 'true', 'success',
                    'success', 'success', row, suite, platform))
                for invalid in (None, dict(row, platform='other'), dict(row, result='failure'),
                                dict(row, result='cancelled'), dict(row, identity={}),
                                dict(row, suite='other'), dict(row, schema_version=2)):
                    self.assertFalse(classifier.conclude(plan, 'true', 'success',
                        'success', 'success', invalid, suite, platform))
                self.assertFalse(classifier.conclude(dict(plan, identity={}), 'true',
                    'success', 'success', 'success', row, suite, platform))

    def test_unavailable_refs_and_shallow_history_fall_back_full(self):
        with tempfile.TemporaryDirectory() as directory:
            def git(*args):
                return subprocess.check_output(['git', '-C', directory, *args]).decode().strip()
            git('init', '-q')
            git('config', 'user.email', 'fixture@example.invalid')
            git('config', 'user.name', 'Fixture')
            git('config', 'commit.gpgsign', 'false')
            root = Path(directory)
            (root / 'README.md').write_text('base')
            git('add', '.')
            git('commit', '-qm', 'base')
            base = git('rev-parse', 'HEAD')
            (root / 'README.md').write_text('head')
            git('commit', '-qam', 'head')
            head = git('rev-parse', 'HEAD')
            (root / '.git/shallow').write_text(head + '\n')
            for revision in (base, '0' * 40):
                result = json.loads(subprocess.check_output(['python',
                    str(ROOT / 'scripts/classify-ci-changes.py'), '--base', revision,
                    '--head', head], cwd=directory))
                self.assertTrue(result['full'])
                self.assertEqual(result['reason'], 'unverifiable_revision_relation')
        workflow = (ROOT / '.github/workflows/ci.yml').read_text()
        block = workflow.split('\n  classify:\n')[1].split('\n  repository:')[0]
        self.assertIn('fetch-depth: 1', block)
        self.assertNotIn('fetch-depth: 0', block)
        self.assertIn('--fetch --output', block)

    def test_outcome_artifact_is_bound_to_run_and_plan(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            env = dict(os.environ, GITHUB_EVENT_NAME='pull_request', GITHUB_SHA='a' * 40,
                       GITHUB_REPOSITORY='fixture/repo', GITHUB_RUN_ID='123',
                       GITHUB_RUN_ATTEMPT='2', SELECTED='false', RESULT='skipped',
                       CLASSIFICATION='success', REPOSITORY='success')
            with patch.dict(os.environ, env):
                plan = classifier.plan([dict(path='docs/x.md', status='M')], 'b' * 40, 'c' * 40)
            (root / 'plan.json').write_text(json.dumps(plan))
            command = ['python', str(ROOT / 'scripts/classify-ci-changes.py'), '--conclude',
                       '--suite', 'verify', '--platform', 'linux', '--plan', str(root / 'plan.json'),
                       '--output', str(root / 'outcome.json')]
            for result, passed in [('skipped', True), ('failure', False), ('cancelled', False)]:
                process = subprocess.run(command, env=dict(env, RESULT=result), capture_output=True)
                self.assertEqual(process.returncode == 0, passed, process.stderr)
                evidence = json.loads((root / 'outcome.json').read_text())
                self.assertEqual(evidence['conclusion'], 'success' if passed else 'failure')
                self.assertEqual(evidence['reason'], 'not_applicable_to_docs')
                self.assertEqual(evidence['identity']['run_id'], '123')
                self.assertEqual(evidence['result'], result)


if __name__ == '__main__':
    unittest.main()
