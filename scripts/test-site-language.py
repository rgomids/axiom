#!/usr/bin/env python3
"""Offline Chromium regression tests for landing-page language switching.

Install test-only dependencies: pip install playwright==1.63.0
Then: python -m playwright install chromium
Run: python scripts/test-site-language.py [repository-root]
All browser requests are fulfilled from site/ or aborted; no public site is used.
"""

from pathlib import Path
import sys
import unittest
from urllib.parse import urlparse

from playwright.sync_api import sync_playwright


ROOT = Path(sys.argv.pop(1)).resolve() if len(sys.argv) > 1 else Path(__file__).resolve().parents[1]


class LanguageTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.playwright = sync_playwright().start()
        cls.browser = cls.playwright.chromium.launch()

    @classmethod
    def tearDownClass(cls):
        cls.browser.close()
        cls.playwright.stop()

    def setUp(self):
        self.context = self.browser.new_context()
        self.page = self.context.new_page()

        def serve(route):
            url = urlparse(route.request.url)
            name = url.path.lstrip('/')
            if url.netloc != 'axiom.test' or name not in (
                'index.html', 'styles.css', 'lang.js', 'diagram.js', 'skills.js'
            ):
                route.abort()
            elif name == 'lang.js':
                # Set test conditions before the real production script runs.
                route.fulfill(body='', content_type='text/javascript')
            else:
                route.fulfill(path=str(ROOT / 'site' / name))

        self.page.route('**/*', serve)

    def tearDown(self):
        self.context.close()

    def load(self, stored='en', blocked=False, width=1280):
        self.page.set_viewport_size({'width': width, 'height': 900})
        if blocked:
            self.page.add_init_script("""
                Object.defineProperty(window, 'localStorage', {
                    get() { throw new Error('storage blocked'); }
                });
            """)
        else:
            self.page.add_init_script(f"localStorage.setItem('axiom-lang', '{stored}');")
        self.page.goto('https://axiom.test/index.html')

    def start_language(self):
        self.page.add_script_tag(path=str(ROOT / 'site/lang.js'))

    def choose(self, lang):
        self.page.locator('#lang-toggle').click()
        self.page.locator(f'.lang-option[data-lang="{lang}"]').click()

    def test_translations_round_trip_preserves_markup_and_navigation(self):
        for width, stored in ((1280, 'en'), (375, 'pt')):
            with self.subTest(width=width, stored=stored):
                self.load(stored=stored, width=width)
                self.page.evaluate("""
                    window.originalText = [...document.querySelectorAll('[data-pt]')]
                        .map(el => el.textContent);
                    window.originalMarkup = [...document.querySelectorAll(
                        '.pill-dot, .gradient-text, .callout a, .callout code, '
                        + '.skill-invoke code, .track-disclaimer strong, .flow-process [aria-hidden]'
                    )];
                    window.originalLinks = originalMarkup.filter(el => el.tagName === 'A')
                        .map(el => el.getAttribute('href'));
                """)
                self.start_language()
                for lang in ('pt', 'en', 'pt', 'en'):
                    self.choose(lang)
                    self.assertEqual(self.page.locator('html').get_attribute('lang'),
                                     'pt-BR' if lang == 'pt' else 'en')
                    self.assertTrue(self.page.evaluate("""lang =>
                        [...document.querySelectorAll('[data-pt]')].every((el, i) =>
                            el.textContent === (lang === 'pt' ? el.getAttribute('data-pt') : originalText[i]))
                    """, lang))
                    self.assertTrue(self.page.evaluate("""() =>
                        originalMarkup.every(el => el.isConnected) &&
                        originalMarkup.filter(el => el.tagName === 'A').every((el, i) =>
                            el.getAttribute('href') === originalLinks[i])
                    """))
                    self.assertEqual(self.page.locator('#lang-toggle').get_attribute('aria-expanded'), 'false')
                    self.assertEqual(self.page.locator(f'.lang-option[data-lang="{lang}"]').get_attribute('aria-checked'), 'true')
                    self.assertEqual(self.page.evaluate("localStorage.getItem('axiom-lang')"), lang)

    def test_translation_attributes_cannot_create_executable_markup(self):
        self.load()
        payload = '<img src="data:," onerror="window.__languageProbe=1"><svg onload="window.__languageProbe=1"></svg>'
        self.page.evaluate("""payload => {
            window.__languageProbe = 0;
            document.querySelectorAll('[data-pt]').forEach(el => {
                el.setAttribute('data-pt', payload);
                el.setAttribute('data-en', payload);
            });
            // The exact former source, including the cached English attribute.
            document.querySelector('.hero-title').setAttribute('data-pt-html', payload);
            document.querySelector('.hero-title').setAttribute('data-en-html', payload);
        }""", payload)
        self.start_language()
        for lang in ('pt', 'en', 'pt'):
            self.choose(lang)
            self.page.evaluate('() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)))')
            self.assertEqual(self.page.evaluate('window.__languageProbe'), 0)
            self.assertEqual(self.page.locator('img[src="data:,"], svg[onload]').count(), 0)
            self.assertTrue(self.page.evaluate("""payload =>
                [...document.querySelectorAll('[data-pt]')].every(el => el.textContent === payload)
            """, payload))

    def test_metacharacters_and_empty_translation_stay_text(self):
        self.load(stored='pt')
        values = ['', '<>&"\' &lt;svg onload=alert(1)&gt;', 'ação → 日本語']
        self.page.evaluate("""values => {
            window.edgeElements = [...document.querySelectorAll('[data-pt]')].slice(0, values.length);
            window.edgeOriginals = edgeElements.map(el => el.textContent);
            edgeElements.forEach((el, i) => el.setAttribute('data-pt', values[i]));
        }""", values)
        self.start_language()
        self.assertEqual(self.page.evaluate('edgeElements.map(el => el.textContent)'), values)
        self.choose('en')
        self.assertTrue(self.page.evaluate('edgeElements.every((el, i) => el.textContent === edgeOriginals[i])'))

    def test_blocked_storage_and_keyboard_menu(self):
        self.load(blocked=True)
        self.start_language()
        self.assertEqual(self.page.locator('html').get_attribute('lang'), 'en')
        self.page.locator('#lang-toggle').focus()
        self.page.keyboard.press('ArrowDown')
        self.assertEqual(self.page.locator(':focus').get_attribute('data-lang'), 'en')
        self.page.keyboard.press('ArrowDown')
        self.page.keyboard.press('Enter')
        self.assertEqual(self.page.locator('html').get_attribute('lang'), 'pt-BR')
        self.assertEqual(self.page.locator(':focus').get_attribute('id'), 'lang-toggle')
        self.page.keyboard.press('ArrowUp')
        self.page.keyboard.press('Escape')
        self.assertEqual(self.page.locator('#lang-toggle').get_attribute('aria-expanded'), 'false')


if __name__ == '__main__':
    unittest.main()
