#!/usr/bin/env python3
"""Exercise the real gate with isolated Go modules, without Provider calls."""
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
from unittest import mock

ROOT = Path(__file__).resolve().parent.parent


class GoQuality(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        (self.root / 'scripts').mkdir()
        shutil.copy2(ROOT / 'scripts/check-go-quality.sh', self.root / 'scripts')
        subprocess.run(['git', 'init', '-q', str(self.root)], check=True)
        (self.root / 'go.mod').write_text('module quality.example/fixture\n\ngo 1.26.0\n')
        self.source('package fixture\n\nfunc Good() int { return 1 }\n')

    def source(self, text, name='quality.go'):
        (self.root / name).write_text(text)

    def gate(self, mode='all'):
        return subprocess.run(['bash', str(self.root / 'scripts/check-go-quality.sh'), mode],
                              capture_output=True, text=True, timeout=90)

    def assert_failure(self, mode, diagnostic):
        result = self.gate(mode)
        self.assertNotEqual(result.returncode, 0, result.stdout)
        self.assertIn(diagnostic, result.stdout + result.stderr)
        return result

    def test_clean_module_passes_without_mutation(self):
        before = {p.name: p.read_bytes() for p in self.root.glob('go.*')}
        result = self.gate()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('staticcheck 2026.2.1 (0.8.1)', result.stdout)
        self.assertEqual(before, {p.name: p.read_bytes() for p in self.root.glob('go.*')})

    def test_unformatted_new_file_with_space_fails_read_only(self):
        self.source('package fixture\nfunc Added( ) { }\n', 'new file.go')
        before = (self.root / 'new file.go').read_bytes()
        self.assert_failure('format', 'new file.go')
        self.assertEqual(before, (self.root / 'new file.go').read_bytes())

    def test_invalid_go_syntax_fails(self):
        self.source('package fixture\nfunc broken(\n')
        self.assert_failure('format', 'gofmt could not parse')

    def test_untidy_module_fails_read_only(self):
        module = self.root / 'go.mod'
        module.write_text(module.read_text() + '\nrequire go.yaml.in/yaml/v3 v3.0.5\n')
        before = module.read_bytes()
        self.assert_failure('tidy', 'module tidiness')
        self.assertEqual(before, module.read_bytes())

    def test_correctness_finding_has_location_and_code(self):
        self.source('package fixture\n\nimport "regexp"\n\nfunc Broken() { regexp.MustCompile("[") }\n')
        result = self.assert_failure('staticcheck', 'SA1000')
        self.assertIn('quality.go:5:', result.stdout)

    def test_unused_private_code_fails(self):
        self.source('package fixture\n\nfunc unused() {}\n')
        self.assert_failure('staticcheck', 'U1000')

    def test_style_does_not_block(self):
        self.source('package fixture\n\nfunc Good_name() {}\n')
        result = self.gate()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_unpinned_analyzer_is_refused(self):
        fake = self.root / 'fake-staticcheck'
        fake.write_text('#!/usr/bin/env bash\nprintf "staticcheck wrong-version\\n"\n')
        fake.chmod(0o755)
        with mock.patch.dict(os.environ, {'STATICCHECK': str(fake)}):
            self.assert_failure('staticcheck', 'expected Staticcheck v0.8.1')


class WorkflowContract(unittest.TestCase):
    def test_suppressions_are_local_and_documented(self):
        import re
        listing = subprocess.run(['git', '-C', str(ROOT), 'ls-files', '-z',
                                  '--cached', '--others', '--exclude-standard', '--', '*.go'],
                                 capture_output=True, check=True).stdout.decode()
        for name in set(listing.split('\0')) - {''}:
            path = ROOT / name
            if not path.is_file():
                continue
            for line_number, line in enumerate(path.read_text().splitlines(), 1):
                directive = re.match(r'\s*//\s*lint:(\S+)(.*)', line)
                if directive is None:
                    continue
                with self.subTest(file=name, line=line_number):
                    self.assertEqual(directive[1], 'ignore', 'Only local lint:ignore is allowed')
                    self.assertRegex(directive[2], r'^\s+[A-Z]+\d+(?:,[A-Z]+\d+)*\s+\S.*',
                                     'A local exception needs explicit check IDs and a reason')

    def test_required_parallel_bounded_gate_and_existing_checks(self):
        import json
        workflow = (ROOT / '.github/workflows/ci.yml').read_text()
        quality = workflow.split('  go-quality-suite:\n', 1)[1].split('\n  verify-suite:', 1)[0]
        self.assertIn('name: go-quality', quality)
        self.assertIn('timeout-minutes: 10', quality)
        self.assertIn('needs: classify', quality)
        self.assertNotIn('continue-on-error:', quality)
        self.assertIn("if: needs.classify.outputs.go == 'true'", quality)
        self.assertIn('persist-credentials: false', quality)
        for command in ('format', 'tidy', 'staticcheck'):
            self.assertIn(f'./scripts/check-go-quality.sh {command}', quality)
        self.assertIn('staticcheck@v0.8.1', quality)
        for command in ('go test -race ./...', 'go test ./... -timeout 10m',
                        'go vet ./...', 'go build ./...', 'go mod verify',
                        './scripts/validate-repository.sh .', './scripts/test-release-pipeline.sh',
                        './scripts/test-upgrade-journeys.sh'):
            self.assertIn(command, workflow)
        rules = json.loads((ROOT / '.github/rulesets/main.json').read_text())
        checks = next(r['parameters']['required_status_checks'] for r in rules['rules']
                      if r['type'] == 'required_status_checks')
        self.assertIn({'context': 'go-quality', 'integration_id': 15368}, checks)


if __name__ == '__main__':
    unittest.main()
