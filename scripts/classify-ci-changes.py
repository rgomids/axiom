#!/usr/bin/env python3
"""Closed path classification for PR CI; unknown/mixed inputs run every suite."""
import argparse
import json
import os
import subprocess
import sys
from pathlib import Path

SUITES = ('go', 'verify', 'release', 'upgrade')


def identity():
    return {key: os.environ.get(env, '') for key, env in (
        ('event', 'GITHUB_EVENT_NAME'), ('repository', 'GITHUB_REPOSITORY'),
        ('tested_sha', 'GITHUB_SHA'), ('run_id', 'GITHUB_RUN_ID'),
        ('run_attempt', 'GITHUB_RUN_ATTEMPT'))}


def category(path):
    # Security and execution policy takes precedence over narrower surfaces.
    if path.startswith(('.github/workflows/', '.github/rulesets/', '.agents/policies/')) or path in (
            'AGENTS.md', 'CLAUDE.md', 'SECURITY.md', '.github/CODEOWNERS',
            'docs/security/repository-security.md'):
        return 'sensitive'
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
    full = force_full or not categories or bool({'unknown', 'sensitive'} & set(categories)) or len(categories) > 1
    selected = categories[0] if not full else 'full'
    return dict(categories=categories, full=full,
                go=full or selected in ('go', 'installer'),
                verify=full or selected in ('go', 'installer'),
                release=full or selected in ('installer', 'release'),
                upgrade=full or selected in ('go', 'installer'))


def changes(base, head, fetch=False):
    # Bounded fetch: never download the complete history for classification.
    if fetch:
        import re
        if not all(re.fullmatch(r'[0-9a-f]{40}', sha or '') for sha in (base, head)):
            raise ValueError('invalid revision identity')
        subprocess.run(['git', 'fetch', '--no-tags', '--depth=64', 'origin', base, head],
                       check=True, capture_output=True, timeout=90)
    # Prove ancestry before permitting partial routing; insufficient shallow
    # history is conservative FULL, even when endpoint trees are available.
    subprocess.run(['git', 'merge-base', base, head], check=True, capture_output=True)
    raw = subprocess.check_output(['git', 'diff', '--no-renames', '--name-status',
                                   '-z', base, head, '--'])
    fields = raw.split(b'\0')[:-1]
    if len(fields) % 2:
        raise ValueError('malformed Git path status')
    return [dict(status=fields[i].decode('ascii'),
                 path=fields[i + 1].decode('utf-8', errors='surrogateescape'))
            for i in range(0, len(fields), 2)]


def plan(entries, base=None, head=None, force_full=False, fallback=None):
    paths = [entry['path'] for entry in entries]
    result = classify(paths, force_full or bool(fallback))
    reason = (fallback or ('manual_dispatch' if force_full else
              'sensitive_paths' if 'sensitive' in result['categories'] else
              'unknown_paths' if 'unknown' in result['categories'] else
              'empty_diff' if not paths else
              'mixed_categories' if len(result['categories']) > 1 else 'single_category'))
    return dict(schema_version=1, identity=dict(identity(), base=base, head=head),
                paths=paths, changes=[dict(entry, category=category(entry['path'])) for entry in entries],
                reason=reason, suites={key: dict(selected=result[key],
                    reason=reason if result[key] else 'not_applicable_to_' + result['categories'][0])
                    for key in SUITES}, **result)


def row_matches(record, suite, platform):
    return record == dict(schema_version=1, identity=identity(), suite=suite,
                          platform=platform, result='success')


def read_record(path):
    try:
        record = json.loads(Path(path).read_text(encoding='utf-8'))
        return record if isinstance(record, dict) else {}
    except (OSError, ValueError, TypeError):
        return {}


def valid_plan(record):
    """Validate the complete v1 shape and recompute routing/N/A semantics."""
    try:
        if type(record['schema_version']) is not int or record['schema_version'] != 1:
            return False
        if any(type(record[key]) is not bool for key in ('full', *SUITES)):
            return False
        if any(type(record['suites'][key]['selected']) is not bool for key in SUITES):
            return False
        entries = [dict(path=item['path'], status=item['status']) for item in record['changes']]
        if not all(isinstance(e['path'], str) and e['path'] and
                   e['status'] in ('A', 'M', 'D', 'T', 'U', 'X', 'B') for e in entries):
            return False
        event = identity()['event']
        if event == 'workflow_dispatch' and record['reason'] != 'manual_dispatch':
            return False
        if event == 'pull_request' and record['reason'] == 'manual_dispatch':
            return False
        expected = plan(entries, record['identity']['base'], record['identity']['head'],
                        record['reason'] == 'manual_dispatch',
                        'unverifiable_revision_relation' if record['reason'] == 'unverifiable_revision_relation' else None)
        return record == expected
    except (KeyError, TypeError, ValueError, AttributeError):
        return False


def conclude(plan_record, selected, result, classification, repository, row=None,
             suite=None, platform=None):
    return (valid_plan(plan_record) and
            all(plan_record.get('identity', {}).get(k) == v for k, v in identity().items()) and
            suite in SUITES and selected == str(plan_record.get('suites', {}).get(suite, {}).get('selected')).lower() and
            classification == repository == 'success' and
            ((selected == 'true' and result == 'success' and
              (platform is None or row_matches(row, suite, platform))) or
             (selected == 'false' and result == 'skipped')))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--base')
    parser.add_argument('--head')
    parser.add_argument('--full', action='store_true')
    parser.add_argument('--fetch', action='store_true')
    parser.add_argument('--output')
    parser.add_argument('--record-row', action='store_true')
    parser.add_argument('--conclude', action='store_true')
    parser.add_argument('--suite', choices=SUITES)
    parser.add_argument('--platform')
    parser.add_argument('--plan')
    parser.add_argument('--row')
    args = parser.parse_args()
    if args.record_row:
        if not args.suite or not args.platform or not args.output:
            parser.error('row recording requires suite, platform and output')
        if args.platform != {'linux': 'linux', 'darwin': 'macos', 'win32': 'windows'}.get(sys.platform):
            raise SystemExit('platform identity does not match native runner')
        Path(args.output).write_text(json.dumps(dict(schema_version=1, identity=identity(),
            suite=args.suite, platform=args.platform, result='success')), encoding='utf-8')
        return
    if args.conclude:
        record = read_record(args.plan)
        row = read_record(args.row) if args.row else None
        selected, result, classification, repository = [os.environ.get(key, '') for key in
            ('SELECTED', 'RESULT', 'CLASSIFICATION', 'REPOSITORY')]
        passed = conclude(record, selected, result, classification, repository,
                          row, args.suite, args.platform)
        validated = valid_plan(record)
        plan_identity = record.get('identity') if isinstance(record.get('identity'), dict) else {}
        evidence = dict(schema_version=1, identity=dict(identity(),
            base=plan_identity.get('base'), head=plan_identity.get('head')), suite=args.suite,
            platform=args.platform, selected=selected, result=result,
            classification=classification, repository=repository,
            reason=record['suites'][args.suite]['reason'] if validated else 'classification_unavailable', row=row,
            conclusion='success' if passed else 'failure')
        Path(args.output).write_text(json.dumps(evidence, indent=2), encoding='utf-8')
        if os.environ.get('GITHUB_STEP_SUMMARY'):
            with open(os.environ['GITHUB_STEP_SUMMARY'], 'a', encoding='utf-8') as summary:
                summary.write('CI outcome: `' + json.dumps(evidence) + '`\n')
        if not passed:
            raise SystemExit('required CI conclusion failed closed')
        return
    entries = []
    fallback = None
    if not args.full:
        if not args.base or not args.head:
            parser.error('--base and --head are required unless --full')
        try:
            entries = changes(args.base, args.head, args.fetch)
        except (subprocess.SubprocessError, ValueError):
            fallback = 'unverifiable_revision_relation'
    result = plan(entries, args.base, args.head, args.full, fallback)
    print(json.dumps(result, ensure_ascii=True))
    if args.output:
        Path(args.output).write_text(json.dumps(result, indent=2), encoding='utf-8')
    if os.environ.get('GITHUB_OUTPUT'):
        with open(os.environ['GITHUB_OUTPUT'], 'a', encoding='utf-8') as output:
            for key in ('go', 'verify', 'release', 'upgrade'):
                output.write(f'{key}={str(result[key]).lower()}\n')
    if os.environ.get('GITHUB_STEP_SUMMARY'):
        with open(os.environ['GITHUB_STEP_SUMMARY'], 'a', encoding='utf-8') as summary:
            summary.write('CI classification: `' + json.dumps(result) + '`\n')


if __name__ == '__main__':
    main()
