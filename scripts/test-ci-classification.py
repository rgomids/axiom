#!/usr/bin/env python3
"""Deterministic path fixtures and fail-closed required-check contracts."""
import importlib.util
import json
import os
import re
from pathlib import Path
import subprocess
import shutil
import tempfile
import unittest

ROOT = Path(__file__).resolve().parent.parent
spec = importlib.util.spec_from_file_location('classifier', ROOT / 'scripts/classify-ci-changes.py')
classifier = importlib.util.module_from_spec(spec)
spec.loader.exec_module(classifier)


class Routing(unittest.TestCase):
    def test_fixture_matrix(self):
        fixtures = [
            (['docs/development/guide.md', 'README.md'], []),
            (['site/index.html', 'docs/assets/logo.svg'], []),
            (['.agents/policies/security.md'], []),
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
                self.assertIn('[[ "$CLASSIFICATION" == success && "$REPOSITORY" == success ]]', block)
                self.assertIn('[[ "$RESULT" == success ]]', block)
                self.assertIn('[[ "$SELECTED" == false && "$RESULT" == skipped ]]', block)
                self.assertNotIn('continue-on-error', block)
                command = block.split('        run: |\n', 1)[1]
                command = '\n'.join(line[10:] for line in command.splitlines())
                bash = 'C:/Program Files/Git/bin/bash.exe' if os.name == 'nt' else shutil.which('bash')
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
                    with tempfile.NamedTemporaryFile() as summary:
                        env = dict(os.environ, CLASSIFICATION=classification, REPOSITORY=repository,
                                   SELECTED=selected, RESULT=result, GITHUB_STEP_SUMMARY=summary.name)
                        process = subprocess.run([bash, '-e', '-c', command], env=env,
                                                 capture_output=True)
                        self.assertEqual(process.returncode == 0, passed,
                                         (job, classification, repository, selected, result, process.stderr))
                suite = re.split(r'\n  [a-z][a-z-]*:\n', workflow.split('\n  ' + job + '-suite:\n')[1], 1)[0]
                self.assertIn("if: needs.classify.outputs." + flag + " == 'true'", suite)
        self.assertIn('python3 scripts/classify-ci-changes.py --full', workflow)
        self.assertNotIn('  push:', workflow)
        self.assertIn('python3 scripts/test-ci-classification.py', workflow)


if __name__ == '__main__':
    unittest.main()
