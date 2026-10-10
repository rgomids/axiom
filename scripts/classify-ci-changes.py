#!/usr/bin/env python3
"""Closed path classification for PR CI; unknown/mixed inputs run every suite."""
import argparse
import json
import os
import subprocess


def category(path):
    if path.startswith(('site/', 'docs/assets/')) or path in (
            'scripts/validate-landing-page.sh', 'scripts/test-site-language.py',
            '.github/workflows/deploy-landpage.yml'):
        return 'site'
    if path.startswith(('internal/compatibility/', 'internal/installation/')) or path in (
            'install.sh', 'install.ps1') or path.startswith(('scripts/install',
            'scripts/test-install', 'scripts/test-windows', 'scripts/test-upgrade')):
        return 'installer'
    if path.startswith(('cmd/', 'internal/')) or path in ('go.mod', 'go.sum'):
        return 'go'
    if path.startswith(('scripts/release-', 'scripts/test-release-',
                        'scripts/verify-release-',
                        'scripts/publish-release', 'scripts/verify-prepared-release')):
        return 'release'
    if path.startswith('docs/') and path.endswith('.md') or path in ('README.md', 'CHANGELOG.md'):
        return 'docs'
    if path.startswith(('.agents/', '.claude/', '.github/rulesets/')) or path in (
            'AGENTS.md', 'CLAUDE.md', 'SECURITY.md', '.gitignore'):
        return 'repository'
    # Scripts/workflows are cross-cutting unless explicitly bounded above.
    return 'unknown'


def classify(paths, force_full=False):
    categories = sorted({category(p) for p in paths})
    full = force_full or not categories or 'unknown' in categories or len(categories) > 1
    selected = categories[0] if not full else 'full'
    return dict(categories=categories, full=full,
                go=full or selected in ('go', 'installer'),
                verify=full or selected in ('go', 'installer'),
                release=full or selected in ('installer', 'release'),
                upgrade=full or selected in ('go', 'installer'))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--base')
    parser.add_argument('--head')
    parser.add_argument('--full', action='store_true')
    args = parser.parse_args()
    paths = []
    if not args.full:
        if not args.base or not args.head:
            parser.error('--base and --head are required unless --full')
        # No rename detection: both deleted and added paths must affect routing.
        raw = subprocess.check_output(['git', 'diff', '--no-renames', '--name-only',
                                       '-z', args.base, args.head, '--'])
        paths = [p.decode('utf-8', errors='surrogateescape') for p in raw.split(b'\0') if p]
    result = classify(paths, args.full)
    print(json.dumps(dict(paths=paths, **result), ensure_ascii=True))
    if os.environ.get('GITHUB_OUTPUT'):
        with open(os.environ['GITHUB_OUTPUT'], 'a', encoding='utf-8') as output:
            for key in ('go', 'verify', 'release', 'upgrade'):
                output.write(f'{key}={str(result[key]).lower()}\n')
    if os.environ.get('GITHUB_STEP_SUMMARY'):
        with open(os.environ['GITHUB_STEP_SUMMARY'], 'a', encoding='utf-8') as summary:
            summary.write('CI classification: `' + json.dumps(result) + '`\n')


if __name__ == '__main__':
    main()
