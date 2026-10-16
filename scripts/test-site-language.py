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
        self.js_enabled = True

        def serve(route):
            url = urlparse(route.request.url)
            path = (ROOT / 'site' / url.path.lstrip('/')).resolve()
            if url.netloc != 'axiom.test' or not path.is_relative_to(ROOT / 'site') or not path.is_file():
                route.abort()
            elif self.defer_app and path.name in ('app.js', 'storytelling.js', 'story-state.js'):
                route.fulfill(body='', content_type='text/javascript')
            else:
                route.fulfill(path=str(path))
        self.serve = serve
        self.page.route('**/*', serve)

    def tearDown(self):
        self.context.close()

    def load(self, width=1280, blocked=False):
        self.page.set_viewport_size({'width': width, 'height': 900})
        if blocked:
            self.page.add_init_script("Object.defineProperty(window,'localStorage',{get(){throw Error('blocked')}})")
        self.page.goto('https://axiom.test/index.html')
        if self.js_enabled and not self.defer_app:
            self.page.wait_for_function("document.querySelector('.storytelling').hasAttribute('data-story-progress')")
            self.page.evaluate('() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)))')

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

    def seek_story(self, progress):
        self.page.evaluate("""progress => {
            const story=document.querySelector('.storytelling');
            const top=story.getBoundingClientRect().top+scrollY;
            scrollTo({top:top+(story.offsetHeight-innerHeight)*progress/4,behavior:'instant'});
        }""", progress)
        self.page.evaluate('() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)))')
        self.page.wait_for_function("expected=>Math.abs(Number(document.querySelector('.storytelling').dataset.storyProgress)-expected)<.01", arg=progress, timeout=3000)

    def story_snapshot(self):
        return self.page.evaluate("""() => [...document.querySelectorAll(
            '.story-stage [data-story-idea],.story-stage [data-story-work],.story-stage [data-story-path],.story-stage [data-story-axiom]'
        )].map(el=>({transform:getComputedStyle(el).transform,opacity:getComputedStyle(el).opacity,width:getComputedStyle(el).width,height:getComputedStyle(el).height,d:el.getAttribute('d')}))""")

    def test_native_story_reuses_entities_and_reverses_deterministically(self):
        self.page.emulate_media(reduced_motion='no-preference')
        self.load()
        self.page.evaluate("window.storyEntities=[...document.querySelectorAll('.story-stage [data-story-work],.story-stage [data-story-idea]')]")
        self.assertEqual(self.page.locator('.story-stage [data-story-idea]').count(), 1)
        self.assertEqual(self.page.locator('.story-stage [data-story-work]').count(), 6)
        self.assertEqual(self.page.locator('.story-stage [data-story-path]').count(), 6)
        self.page.evaluate("() => {window.nativeScrollEnded = new Promise(resolve => addEventListener('scrollend',resolve,{once:true}));}")
        self.page.keyboard.press('PageDown')
        self.page.wait_for_function('scrollY > 0')
        self.page.evaluate("Promise.race([nativeScrollEnded,new Promise((_,reject)=>setTimeout(()=>reject(Error('native scroll did not end')),3000))])")
        snapshots = {}
        for progress in [index / 6 for index in range(25)]:
            self.seek_story(progress)
            snapshots[progress] = self.story_snapshot()
            self.assertTrue(self.page.evaluate('storyEntities.every(el=>el.isConnected)'))
            hub_opacity = self.page.locator('[data-story-axiom]').evaluate('el=>Number(getComputedStyle(el).opacity)')
            if progress == 1:
                self.assertLess(hub_opacity, .01)
            if progress == 2:
                self.assertGreater(hub_opacity, .9)
            if progress == 22 / 6:
                self.assertGreater(self.page.locator('[data-story-decision]').evaluate('el=>Number(getComputedStyle(el).opacity)'), .9)
                self.assertEqual(self.page.locator('[data-story-decision] li').count(), 5)
                self.assertEqual(self.page.locator('[data-story-decision] .decision-choices>span').count(), 2)
            if progress == 4:
                self.assertGreater(self.page.locator('[data-story-idea]').evaluate('el=>Number(getComputedStyle(el).opacity)'), .9)
                decision = self.page.locator('[data-story-decision]').text_content().lower()
                self.assertIn('aceitar', decision)
                self.assertIn('retrabalho', decision)
            self.choose('en')
            self.assertTrue(self.page.locator('.story-stage').evaluate("el=>[...el.querySelectorAll('[data-en]')].every(node=>node.textContent===node.dataset.en)"))
            self.choose('pt-BR')
        for progress in (3, 1, 4, 2, 0):
            self.seek_story(progress)
            self.assertEqual(self.story_snapshot(), snapshots[progress])
        for anchor, scene in (('#fluxo', '4'), ('#proposta', '3')):
            self.page.evaluate("() => {window.nativeScrollEnded = new Promise(resolve => addEventListener('scrollend',resolve,{once:true}));}")
            self.page.locator(f'.header a[href="{anchor}"]').click()
            self.page.evaluate("Promise.race([nativeScrollEnded,new Promise((_,reject)=>setTimeout(()=>reject(Error('native scroll did not end')),3000))])")
            self.assertEqual(self.page.locator('.storytelling').get_attribute('data-story-scene'), scene)
        self.page.set_viewport_size({'width': 390, 'height': 844})
        self.seek_story(1.5)
        mobile_snapshot = self.story_snapshot()
        self.seek_story(3.5)
        self.seek_story(1.5)
        self.assertEqual(self.story_snapshot(), mobile_snapshot)
        self.assertFalse(self.page.evaluate('document.documentElement.scrollWidth>innerWidth'))
        self.page.locator('#motion').click()
        self.assertTrue(self.page.locator('body').evaluate("el=>el.classList.contains('static-motion')"))
        self.assertEqual(self.errors, [])

    def test_no_javascript_has_five_complete_scenes_and_direct_installation(self):
        self.context.close()
        self.js_enabled = False
        self.context = self.browser.new_context(java_script_enabled=False)
        self.page = self.context.new_page()
        self.page.route('**/*', self.serve)
        self.load(width=390)
        scenes = self.page.locator('article[data-story-scene]')
        self.assertEqual(scenes.count(), 5)
        for index in range(5):
            scene = scenes.nth(index)
            self.assertTrue(scene.is_visible())
            self.assertGreater(len(scene.inner_text().strip()), 80)
            self.assertGreater(scene.locator('svg,img').count(), 0)
        for width in (320, 390, 1280):
            self.page.set_viewport_size({'width': width, 'height': 900})
            self.assertFalse(self.page.evaluate('document.documentElement.scrollWidth>innerWidth'))
        self.assertTrue(self.page.locator('#instalar').is_visible())
        self.assertIn('curl -fsSL https://raw.githubusercontent.com/rgomids/axiom/main/scripts/install.sh | sh', self.page.locator('#instalar').inner_text())
        self.assertGreater(self.page.locator('a[href="#instalar"]').count(), 0)
        self.assertGreater(self.page.locator('a[href*="github.com/rgomids/axiom"]').count(), 0)

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
        for width in (320, 390, 1280):
            self.page.set_viewport_size({'width': width, 'height': 900})
            self.assertFalse(self.page.evaluate('document.documentElement.scrollWidth>innerWidth'))
        self.assertEqual(self.page.locator('article[data-story-scene]').count(), 5)
        self.assertEqual(self.page.locator('.story-beat').count(), 25)
        for scene in self.page.locator('article[data-story-scene]').all():
            self.assertTrue(scene.is_visible())
        self.choose('en')
        self.assertIn('Supported hosts:', self.page.locator('.terminal-platform').inner_text())
        self.assertEqual(self.page.locator('[data-os="windows"]').count(), 0)
        self.assertEqual(self.errors, [])


if __name__ == '__main__':
    unittest.main()
