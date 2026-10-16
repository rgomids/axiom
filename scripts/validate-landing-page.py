#!/usr/bin/env python3
"""Offline structural contract for the generated Axiom DS landing snapshot."""
import hashlib
from html.parser import HTMLParser
import json
from pathlib import Path
import re
import sys
from urllib.parse import urlparse

ROOT = Path(sys.argv[1]).resolve() if len(sys.argv) > 1 else Path(__file__).resolve().parents[1]
SITE = ROOT / 'site'


def require(condition, message):
    if not condition:
        raise ValueError(message)


class Document(HTMLParser):
    def __init__(self):
        super().__init__()
        self.ids = set()
        self.references = []
        self.translations = 0
        self.metadata = {}
        self.story_scenes = []
        self.story_beats = []
        self.story_transcripts = 0
        self.story_entities = {name: 0 for name in ('idea', 'work', 'path', 'axiom', 'decision')}


    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if 'id' in attrs:
            require(attrs['id'] not in self.ids, 'duplicate id: ' + attrs['id'])
            self.ids.add(attrs['id'])
        self.translations += 'data-en' in attrs
        if 'data-story-scene' in attrs:
            self.story_scenes.append(attrs['data-story-scene'])
        if attrs.get('class', '').split().count('story-beat'):
            self.story_beats.append(attrs.get('id'))
        if 'static-transcript' in attrs.get('class', '').split():
            self.story_transcripts += 1
        if 'scene-static' in attrs.get('class', '').split():
            require(attrs.get('aria-hidden') == 'true', 'decorative figures must not announce hidden future panels')
        for name in self.story_entities:
            self.story_entities[name] += 'data-story-' + name in attrs
        for name in ('src', 'href'):
            if name in attrs:
                self.references.append(attrs[name])
        if tag == 'meta':
            self.metadata[attrs.get('property', attrs.get('name'))] = attrs.get('content')
        if tag == 'link' and attrs.get('rel') == 'canonical':
            self.metadata['canonical'] = attrs.get('href')


def validate():
    manifest = json.loads((SITE / '.axiom-landing-source.json').read_text(encoding='utf-8'))
    require(manifest['repository'] == 'https://github.com/rgomids/axiom-ds', 'unexpected source repository')
    require(re.fullmatch('[0-9a-f]{40}', manifest['revision']), 'source revision must be a full SHA')
    files = manifest['files']
    actual = {p.relative_to(SITE).as_posix() for p in SITE.rglob('*') if p.is_file()}
    require(actual == set(files) | {'.axiom-landing-source.json'}, 'unowned, missing or stale site files')
    for name, digest in files.items():
        path = SITE / name
        require(path.resolve().is_relative_to(SITE) and not path.is_symlink(), 'unsafe bundle path')
        require(hashlib.sha256(path.read_bytes()).hexdigest() == digest, 'generated file drift: ' + name)
    html = (SITE / 'index.html').read_text(encoding='utf-8')
    doc = Document()
    doc.feed(html)
    for reference in doc.references:
        if reference.startswith('#'):
            require(reference[1:] in doc.ids, 'broken anchor: ' + reference)
        elif not urlparse(reference).scheme:
            require((SITE / reference.split('#')[0]).is_file(), 'broken local resource: ' + reference)
    require(doc.translations >= 40, 'incomplete bilingual content')
    require(doc.story_scenes == ['1', '2', '3', '4', '5'], 'five complete static scenes must remain in narrative order')
    require(doc.story_beats == [f'quadro-{scene}-{beat}' for scene in range(1, 6) for beat in range(1, 6)], 'all 25 storyboard beats must remain complete and ordered')
    require(doc.story_transcripts == 25, 'every storyboard beat must have its own accessible transcript')
    require(doc.story_entities == {'idea': 1, 'work': 6, 'path': 6, 'axiom': 1, 'decision': 1}, 'animated story must retain one idea, six shared work objects/paths, an Axiom hub and a human decision')
    for name in ('storytelling.js', 'story-state.js', 'storyboard-state.js', 'story-frames.js'):
        require(name in files, 'missing story implementation: ' + name)
    for scene in range(1, 6):
        require('cena-' + str(scene) in doc.ids, 'static story scene anchor missing')
    require(doc.metadata.get('canonical') == 'https://rgomids.github.io/axiom/', 'wrong production canonical URL')
    require(doc.metadata.get('og:type') == 'website' and doc.metadata.get('twitter:card') == 'summary_large_image', 'social metadata missing')
    for anchor in ('inicio', 'proposta', 'fluxo', 'instalar', 'fundacao'):
        require(anchor in doc.ids, 'required section missing: ' + anchor)
    require('conceitual' in html.lower() and 'conceptual' in html.lower(), 'story must identify its conceptual visualization in both languages')
    require('retrabalho' in html.lower() and 'rework' in html.lower(), 'human rework alternative missing')
    require('Apache 2.0' in html and 'github.com/rgomids/axiom' in html, 'required project/license links missing')
    scripts = '\n'.join(p.read_text(encoding='utf-8') for p in SITE.glob('*.js'))
    require(not re.search('innerHTML|outerHTML|insertAdjacentHTML|document\\.write', scripts), 'browser scripts must keep translations and commands as text')
    require('prefers-reduced-motion: reduce' in scripts, 'system motion preference missing')
    require('navigator.clipboard' in scripts and 'copy-command' in scripts, 'individual command copy missing')
    require('curl -fsSL https://raw.githubusercontent.com/rgomids/axiom/main/scripts/install.sh | sh' in scripts, 'installer differs from current product docs')
    require('Hosts suportados: macOS arm64 e Linux amd64/arm64.' in scripts, 'Portuguese supported hosts missing')
    require('Supported hosts: macOS arm64 and Linux amd64/arm64.' in scripts, 'English supported hosts missing')
    require(not re.search(r'windows|install\.ps1|powershell|Invoke-RestMethod', html + scripts, re.I), 'unsupported installation path returned')
    require((SITE / 'assets/axiom-logo.png').read_bytes() == (ROOT / 'docs/assets/axiom-logo-white.png').read_bytes(), 'original brand asset was altered')
    print('PASS: generated snapshot hashes, source revision, original brand, links, metadata, bilingual content, motion and installation contracts')


if __name__ == '__main__':
    try:
        validate()
    except (ValueError, KeyError, OSError) as error:
        sys.exit('FAIL: ' + str(error))
