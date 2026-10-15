#!/usr/bin/env python3
"""Offline safety and reproducibility tests for the recurring DS bundle import."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('sync_landing', Path(__file__).with_name('sync-landing-page.py'))
sync_landing = importlib.util.module_from_spec(spec)
spec.loader.exec_module(sync_landing)
REVISION = '1' * 40


class SyncTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.base = Path(self.temp.name)
        self.source = self.base / 'packages/landing/dist'
        self.target = self.base / 'site'
        self.source.mkdir(parents=True)
        self.target.mkdir()
        for name in ('index.html', 'app.js', 'storytelling.js', 'story-state.js', 'tokens.css'):
            (self.source / name).write_text(name, encoding='utf-8')

    def tearDown(self):
        self.temp.cleanup()

    def run_sync(self, revision=REVISION, dirty=''):
        with patch.object(sync_landing.subprocess, 'check_output', side_effect=[str(self.base), REVISION, dirty]):
            sync_landing.sync(self.source, revision, self.target)

    def test_reproducible_snapshot_and_owned_file_retirement(self):
        (self.target / 'lang.js').write_text('legacy', encoding='utf-8')
        self.run_sync()
        self.assertFalse((self.target / 'lang.js').exists())
        first = (self.target / sync_landing.MANIFEST).read_bytes()
        self.run_sync()
        self.assertEqual(first, (self.target / sync_landing.MANIFEST).read_bytes())
        manifest = json.loads(first)
        self.assertEqual(manifest['revision'], REVISION)
        self.assertEqual(manifest['files']['app.js'], sync_landing.digest(self.source / 'app.js'))

    def test_refuses_unowned_files_dirty_sources_and_wrong_revision(self):
        (self.target / 'app.js').write_text('user work', encoding='utf-8')
        with self.assertRaisesRegex(ValueError, 'unowned'):
            self.run_sync()
        self.assertEqual((self.target / 'app.js').read_text(), 'user work')
        with self.assertRaisesRegex(ValueError, 'uncommitted'):
            self.run_sync(dirty=' M packages/landing/src/app.js')
        with self.assertRaisesRegex(ValueError, 'does not match'):
            self.run_sync(revision='2' * 40)

    def test_refuses_traversal_in_existing_manifest(self):
        (self.target / sync_landing.MANIFEST).write_text(json.dumps({'files': {'../outside': 'invalid'}}))
        with self.assertRaisesRegex(ValueError, 'unsafe existing'):
            self.run_sync()


if __name__ == '__main__':
    unittest.main()
