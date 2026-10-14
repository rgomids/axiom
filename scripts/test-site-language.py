#!/usr/bin/env python3
"""Offline Chromium acceptance for the generated landing; no external requests."""
from pathlib import Path
import sys
import unittest
from urllib.parse import urlparse
from playwright.sync_api import sync_playwright

ROOT = Path(sys.argv.pop(1)).resolve() if len(sys.argv) > 1 else Path(__file__).resolve().parents[1]


class LandingTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.playwright = sync_playwright().start()
        cls.browser = cls.playwright.chromium.launch()

    @classmethod
    def tearDownClass(cls):
        cls.browser.close()
        cls.playwright.stop()

    def setUp(self):
        self.context = self.browser.new_context(reduced_motion='reduce')
        self.page = self.context.new_page()
        self.errors = []
        self.page.on('pageerror', lambda error: self.errors.append(str(error)))
        self.defer_app = False

        def serve(route):
            url = urlparse(route.request.url)
            path = (ROOT / 'site' / url.path.lstrip('/')).resolve()
            if url.netloc != 'axiom.test' or not path.is_relative_to(ROOT / 'site') or not path.is_file():
                route.abort()
            elif self.defer_app and path.name in ('app.js', 'organic-workflow.js'):
                route.fulfill(body='', content_type='text/javascript')
            else:
                route.fulfill(path=str(path))
        self.page.route('**/*', serve)

    def tearDown(self):
        self.context.close()

    def load(self, width=1280, blocked=False):
        self.page.set_viewport_size({'width': width, 'height': 900})
        if blocked:
            self.page.add_init_script("Object.defineProperty(window,'localStorage',{get(){throw Error('blocked')}})")
        self.page.goto('https://axiom.test/index.html')

    def choose(self, language):
        if self.page.locator('html').get_attribute('lang') != language:
            self.page.locator('#language').click()

    def test_bilingual_roundtrip_preserves_markup_and_storage(self):
        self.load()
        self.page.evaluate("window.originalMarkup=[...document.querySelectorAll('h1 br,h2 em,.core-context i')]")
        for language in ('en', 'pt-BR', 'en', 'pt-BR'):
            self.choose(language)
            self.assertEqual(self.page.locator('html').get_attribute('lang'), language)
            self.assertTrue(self.page.evaluate('originalMarkup.every(el=>el.isConnected)'))
            self.assertEqual(self.page.evaluate("localStorage.getItem('axiom-language')"), 'en' if language == 'en' else 'pt')
        self.assertEqual(self.errors, [])

    def test_translation_attributes_cannot_create_executable_markup(self):
        self.defer_app = True
        self.load()
        self.page.evaluate("""() => {
            window.probe=0;
            document.querySelectorAll('[data-en]').forEach(el=>{
                el.dataset.en='<img src="data:," onerror="window.probe=1"><svg onload="window.probe=1"></svg>';
            });
        }""")
        self.page.add_script_tag(path=str(ROOT / 'site/app.js'))
        self.choose('en')
        self.assertEqual(self.page.locator('img[src="data:,"], svg[onload]').count(), 0)
        self.assertEqual(self.page.evaluate('window.probe'), 0)
        self.choose('pt-BR')
        self.assertEqual(self.errors, [])

    def test_keyboard_tabs_and_individual_copy_for_every_step(self):
        self.page.add_init_script("""
            window.copied=[];
            Object.defineProperty(navigator,'clipboard',{value:{writeText:async value=>copied.push(value)}});
        """)
        self.load()
        self.page.locator('#setup-0').focus()
        self.page.keyboard.press('End')
        self.assertEqual(self.page.locator(':focus').get_attribute('id'), 'setup-3')
        for step, count in ((0, 1), (1, 3), (2, 3), (3, 3)):
            self.page.locator(f'#setup-{step}').click()
            commands = self.page.locator('.command-row code').all_text_contents()
            buttons = self.page.locator('.copy-command')
            self.assertEqual(buttons.count(), count)
            self.assertEqual(len(commands), count)
            for index, command in enumerate(commands):
                buttons.nth(index).click()
                self.assertEqual(self.page.evaluate('copied[copied.length-1]'), command)
                self.assertNotIn('\n', command)
            self.assertIn('copiado', self.page.locator('#copy-status').inner_text())
        self.assertEqual(self.errors, [])

    def test_native_scroll_reveals_dark_page_and_complete_workflow(self):
        self.page.emulate_media(reduced_motion='no-preference')
        self.load()
        self.assertTrue(self.page.locator('body').evaluate("el=>el.classList.contains('gold-start')"))
        self.page.locator('#fundacao').scroll_into_view_if_needed()
        self.page.wait_for_function("!document.body.classList.contains('gold-start')")
        self.assertEqual(self.page.locator('[data-node]').count(), 6)
        self.assertEqual(self.page.locator('[data-path]').count(), 6)
        self.page.locator('#motion').click()
        self.assertTrue(self.page.locator('body').evaluate("el=>el.classList.contains('static-motion')"))
        self.assertEqual(self.errors, [])

    def test_copy_denial_selects_only_requested_command(self):
        self.page.add_init_script("Object.defineProperty(navigator,'clipboard',{value:{writeText:async()=>{throw Error('denied')}}})")
        self.load()
        self.page.locator('#setup-1').click()
        self.page.locator('.copy-command').nth(1).click()
        self.assertEqual(self.page.evaluate('getSelection().toString()'), 'axiom help')
        self.assertIn('selecionado', self.page.locator('#copy-status').inner_text())
        self.assertEqual(self.errors, [])

    def test_reduced_motion_mobile_and_blocked_storage(self):
        self.load(width=390, blocked=True)
        self.assertTrue(self.page.locator('body').evaluate("el=>el.classList.contains('static-motion')"))
        self.assertFalse(self.page.evaluate('document.documentElement.scrollWidth>innerWidth'))
        self.choose('en')
        self.assertIn('Supported hosts:', self.page.locator('.terminal-platform').inner_text())
        self.assertEqual(self.page.locator('[data-os="windows"]').count(), 0)
        self.assertEqual(self.errors, [])


if __name__ == '__main__':
    unittest.main()
