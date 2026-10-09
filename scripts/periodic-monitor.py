#!/usr/bin/env python3
"""Bounded scheduled checks. Raw subprocess output never crosses the evidence boundary."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import signal
import subprocess
import sys
import tarfile
import tempfile
import threading
import time
import urllib.error
import urllib.request
from datetime import datetime, timezone

ROOT = Path(__file__).resolve().parent.parent
BASELINE = json.loads((ROOT / 'scripts/periodic-baseline.json').read_text())
BASELINE_DIGEST = hashlib.sha256(json.dumps(BASELINE, sort_keys=True).encode()).hexdigest()
OUTPUT_LIMIT = 1024 * 1024
ARTIFACT_LIMIT = 256 * 1024
DEADLINE = None
CATEGORIES = {'none', 'vulnerability', 'dependency_drift', 'contract_drift',
              'unsupported_version', 'malformed_output', 'missing_binary',
              'nonzero_exit', 'timeout', 'network', 'installation', 'runner',
              'output_limit', 'artifact', 'permission'}
INFRA = {'missing_binary', 'timeout', 'network', 'installation', 'runner',
         'output_limit', 'artifact', 'permission'}
SUITES = {
    'adapters': ['./internal/runtimeadapter', './internal/githubissues'],
    'failures': ['./internal/executiongraph', './internal/graphapplication'],
    'compatibility': ['./internal/compatibility', './internal/runtimeprofile'],
}


def clean_environment(home):
    # No inherited credentials, auth configuration, proxy credentials or Go flags.
    env = {key: os.environ[key] for key in ('PATH', 'SystemRoot', 'WINDIR',
           'TMP', 'TEMP', 'GOCACHE', 'GOMODCACHE') if key in os.environ}
    env.update(HOME=str(home), USERPROFILE=str(home), XDG_CONFIG_HOME=str(home),
               CODEX_HOME=str(home / 'codex'), CLAUDE_CONFIG_DIR=str(home / 'claude'),
               GH_CONFIG_DIR=str(home / 'gh'), GOTOOLCHAIN='local', GOWORK='off',
               GOFLAGS='', CI='true', NO_COLOR='1', GH_PROMPT_DISABLED='1',
               DISABLE_AUTOUPDATER='1', DISABLE_TELEMETRY='1')
    return env


def execute(argv, timeout=30, env=None, cwd=ROOT):
    """Drain both pipes with bounded RAM; kill the entire process group on timeout."""
    started = time.monotonic()
    if DEADLINE is not None:
        remaining = DEADLINE - started
        if remaining <= 1:
            return {'code': None, 'category': 'timeout', 'out': '', 'err': '', 'duration': 0}
        timeout = min(timeout, remaining)
    buffers = [bytearray(), bytearray()]
    overflow = [False]
    try:
        process = subprocess.Popen(argv, cwd=cwd, env=env, stdin=subprocess.DEVNULL,
                                   stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                   start_new_session=os.name != 'nt')
    except FileNotFoundError:
        return {'code': None, 'category': 'missing_binary', 'out': '', 'err': '', 'duration': 0}
    except OSError:
        return {'code': None, 'category': 'runner', 'out': '', 'err': '', 'duration': 0}

    def drain(pipe, target):
        with pipe:
            while chunk := pipe.read(8192):
                available = OUTPUT_LIMIT - len(target)
                target.extend(chunk[:available])
                if len(chunk) > available:
                    overflow[0] = True

    threads = [threading.Thread(target=drain, args=(pipe, target), daemon=True)
               for pipe, target in zip((process.stdout, process.stderr), buffers)]
    for thread in threads:
        thread.start()
    category = 'none'
    try:
        process.wait(timeout=timeout)
        # A child retaining a pipe is also bounded by the command timeout.
        for thread in threads:
            thread.join(max(0, timeout - (time.monotonic() - started)))
        if any(thread.is_alive() for thread in threads):
            raise subprocess.TimeoutExpired(argv, timeout)
    except subprocess.TimeoutExpired:
        category = 'timeout'
        if os.name == 'nt':
            subprocess.run(['taskkill', '/PID', str(process.pid), '/T', '/F'],
                           stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=5)
        else:
            try:
                os.killpg(process.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
        process.wait(timeout=5)
        for thread in threads:
            thread.join(5)
    if category == 'none' and overflow[0]:
        category = 'output_limit'
    if category == 'none' and process.returncode:
        text = bytes(buffers[1] + buffers[0]).decode('utf-8', 'replace').lower()
        category = 'network' if re.search(r'timeout|connection|network|resolve host|econn|tls handshake|502|503', text) else 'nonzero_exit'
    return {'code': process.returncode, 'category': category,
            'out': bytes(buffers[0]).decode('utf-8', 'replace'),
            'err': bytes(buffers[1]).decode('utf-8', 'replace'),
            'duration': round(min(time.monotonic() - started, timeout + 15), 3)}


def fingerprint(check, component, category, contract):
    return hashlib.sha256(f'{check}|{component}|{category}|{contract}'.encode()).hexdigest()


def result(check, component, category='none', *, contract='', duration=0,
           version='unavailable', expected=None, observed=None, reproduction=None):
    status = 'PASS' if category == 'none' else ('INCONCLUSIVE' if category in INFRA else 'FAIL')
    return {'check': check, 'component': component, 'contract': contract,
            'status': status, 'category': category,
            'fingerprint': fingerprint(check, component, category, contract),
            'duration_seconds': duration, 'tool_version': version,
            'baseline': expected, 'observed': observed,
            'reproduction': reproduction or f'python3 scripts/periodic-monitor.py run --layer {"daily" if check in ("security", "tidy", "integrity") else "weekly"} --output /tmp/axiom-periodic',
            'diagnostics': 'manifest.json'}


def parse_version(tool, output):
    patterns = {'gh': r'^gh version (\d+\.\d+\.\d+)(?:\s|$)',
                'codex': r'^codex-cli (\d+\.\d+\.\d+)(?:\s|$)',
                'claude': r'^(\d+\.\d+\.\d+) \(Claude Code\)(?:\s|$)',
                'go': r'^go version (go\d+\.\d+(?:\.\d+)?)(?:\s|$)',
                'govulncheck': r'govulncheck@(v\d+\.\d+\.\d+)'}
    match = re.search(patterns[tool], output.strip())
    return match[1] if match else 'unavailable'


def capabilities(text, expected):
    # Publish ONLY baseline tokens with semantic presence booleans, never raw help.
    normalized = re.sub(r'\x1b\[[0-9;]*m', '', text)
    return {token: bool(re.search(r'(?<![\w-])' + re.escape(token) + r'(?![\w-])', normalized))
            for token in expected}


def check_surface(tool, surface, response, version):
    expected = {token: True for token in surface['capabilities']}
    observed = capabilities(response['out'], expected)
    category = response['category']
    if category == 'none':
        category = 'malformed_output' if not response['out'].strip() or not re.search(r'\busage\b', response['out'], re.I) else (
            'none' if all(observed.values()) else 'contract_drift')
    return result('cli', tool, category, contract=surface['id'],
                  duration=response['duration'], version=version,
                  expected=expected, observed=observed)


def security_category(response):
    # JSON mode may exit 0 even with findings. Never rely only on the exit code.
    if response['category'] not in ('none', 'nonzero_exit'):
        return response['category']
    try:
        decoder = json.JSONDecoder()
        content = response['out'].strip()
        messages = []
        while content:
            message, offset = decoder.raw_decode(content)
            if not isinstance(message, dict):
                return 'malformed_output'
            messages.append(message)
            content = content[offset:].lstrip()
        if not any('config' in m for m in messages):
            return 'malformed_output'
        if any('finding' in m for m in messages):
            return 'vulnerability'
        return 'none' if response['category'] == 'none' else 'nonzero_exit'
    except (ValueError, TypeError):
        return 'malformed_output'


def install(tool, home, env):
    """Exact versions, private runner-local installation, bounded/no install logs."""
    if tool == 'govulncheck':
        env = dict(env, GOBIN=str(home / 'bin'))
        response = execute(['go', 'install', 'golang.org/x/vuln/cmd/govulncheck@' + BASELINE[tool]], 180, env)
        return str(home / 'bin/govulncheck'), response
    if tool in ('codex', 'claude'):
        package = '@openai/codex' if tool == 'codex' else '@anthropic-ai/claude-code'
        response = execute(['npm', 'install', '--prefix', str(home / 'npm'),
                            '--ignore-scripts', '--no-audit', '--no-fund',
                            package + '@' + BASELINE['tools'][tool]['version']], 180, env)
        binary = home / 'npm/node_modules/.bin' / tool
        if tool == 'claude' and response['category'] == 'none':
            # The current wrapper's postinstall replaces an inert .exe stub.
            # Keep lifecycle scripts disabled; use the exact npm-integrity-
            # verified Linux x64 optional binary directly on our runner row.
            binary = home / 'npm/node_modules/@anthropic-ai/claude-code-linux-x64/claude'
            try:
                if binary.is_symlink() or not binary.is_file():
                    raise ValueError('native binary absent')
                with binary.open('rb') as stream:
                    if stream.read(4) != b'\x7fELF':
                        raise ValueError('native binary format')
                binary.chmod(0o700)
            except (OSError, ValueError):
                response = dict(response, category='installation')
        return str(binary), response
    started = time.monotonic()
    if DEADLINE is not None and DEADLINE - started < 65:
        return str(home / 'bin/gh'), {'category': 'timeout', 'duration': 0}
    version = BASELINE['tools']['gh']['version']
    url = f'https://github.com/cli/cli/releases/download/v{version}/gh_{version}_linux_amd64.tar.gz'
    def download(address, limit):
        deadline = min(started + 60, DEADLINE or started + 60)
        content = bytearray()
        with urllib.request.urlopen(address, timeout=10) as stream:
            while True:
                if time.monotonic() >= deadline:
                    raise TimeoutError('download budget')
                chunk = stream.read1(min(65536, limit + 1 - len(content)))
                if not chunk:
                    return bytes(content)
                content.extend(chunk)
                if len(content) > limit:
                    raise ValueError('download limit')
    try:
        # Runner contract is ubuntu-24.04 x64. HTTPS plus release checksum;
        # reject traversal/links and extract just the binary, not an archive tree.
        archive = download(url, 32 * 1024 * 1024)
        checksum_url = f'https://github.com/cli/cli/releases/download/v{version}/gh_{version}_checksums.txt'
        checksums = download(checksum_url, 65536)
        name = f'gh_{version}_linux_amd64.tar.gz'
        if not re.search(r'^' + hashlib.sha256(archive).hexdigest() + r'\s+' + re.escape(name) + r'$', checksums.decode(), re.M):
            raise ValueError('checksum')
        import io
        with tarfile.open(fileobj=io.BytesIO(archive), mode='r:gz') as source:
            member = source.getmember(f'gh_{version}_linux_amd64/bin/gh')
            if not member.isfile() or member.size > 64 * 1024 * 1024:
                raise ValueError('member')
            binary = home / 'bin/gh'
            binary.parent.mkdir(exist_ok=True)
            binary.write_bytes(source.extractfile(member).read())
            binary.chmod(0o700)
        category = 'none'
    except (urllib.error.URLError, TimeoutError, OSError):
        category = 'network'
    except (ValueError, KeyError, tarfile.TarError):
        category = 'installation'
    return str(home / 'bin/gh'), {'category': category, 'duration': round(time.monotonic() - started, 3)}


def daily(home, env):
    checks = []
    go = execute(['go', 'version'], env=env)
    version = parse_version('go', go['out'])
    for check, argv in [('tidy', ['go', 'mod', 'tidy', '-diff']), ('integrity', ['go', 'mod', 'verify'])]:
        response = execute(argv, 120, env)
        category = response['category']
        if category == 'none' and version == 'unavailable':
            category = go['category'] if go['category'] != 'none' else 'malformed_output'
        if category == 'nonzero_exit':
            category = 'dependency_drift'
        checks.append(result(check, 'go', category, version=version,
                             duration=response['duration'], reproduction=' '.join(argv)))
    binary, response = install('govulncheck', home, env)
    if response['category'] != 'none':
        checks.append(result('security', 'govulncheck', response['category'] if response['category'] in INFRA else 'installation'))
        return checks
    response = execute([binary, '-version'], env=env)
    scanner_version = parse_version('govulncheck', response['out'])
    if scanner_version != BASELINE['govulncheck']:
        category = response['category'] if response['category'] != 'none' else 'unsupported_version'
    else:
        response = execute([binary, '-json', './...'], 240, env)
        category = security_category(response)
    checks.append(result('security', 'govulncheck', category, version=scanner_version,
                         duration=response['duration'], reproduction='govulncheck ./... (local only; follow SECURITY.md)'))
    return checks


def parse_go_tests(response):
    tests = []
    allowed = test_names()
    for line in response['out'].splitlines():
        try:
            event = json.loads(line)
        except ValueError:
            continue
        name = event.get('Test', '')
        if event.get('Action') in ('pass', 'fail', 'skip') and name in allowed:
            tests.append({'test': name, 'status': {'pass': 'PASS', 'fail': 'FAIL', 'skip': 'SKIP'}[event['Action']]})
    return tests[:500]


def test_names():
    # Names are repository-owned top-level declarations, never vendor text or
    # dynamic subtest names (which can include raw error/configuration data).
    return {name for packages in SUITES.values() for package in packages
            for path in (ROOT / package).glob('*_test.go')
            for name in re.findall(r'func (Test[A-Za-z0-9_]+)\(', path.read_text(encoding='utf-8'))}


def weekly(home, env):
    checks = []
    for tool, baseline in BASELINE['tools'].items():
        binary, response = install(tool, home, env)
        if response['category'] != 'none':
            category = response['category'] if response['category'] in INFRA else 'installation'
            checks.append(result('cli', tool, category, contract='version'))
            continue
        response = execute([binary, '--version'], env=env)
        version = parse_version(tool, response['out'])
        category = response['category']
        if category == 'none':
            category = 'malformed_output' if version == 'unavailable' else (
                'none' if version == baseline['version'] else 'unsupported_version')
        checks.append(result('cli', tool, category, contract='version', version=version,
                             expected=baseline['version'], observed=version, duration=response['duration']))
        if category != 'none':
            continue
        for surface in baseline['surfaces']:
            response = execute([binary] + surface['args'], env=env)
            checks.append(check_surface(tool, surface, response, version))
    go = execute(['go', 'version'], env=env)
    version = parse_version('go', go['out'])
    for suite, packages in SUITES.items():
        argv = ['go', 'test', '-json', '-count=1', '-timeout=180s'] + packages
        response = execute(argv, 210, env)
        row = result('simulation', 'go', response['category'], contract=suite,
                     version=version, duration=response['duration'], reproduction=' '.join(argv))
        row['tests'] = parse_go_tests(response)
        if row['status'] == 'PASS' and version == 'unavailable':
            category = go['category'] if go['category'] != 'none' else 'malformed_output'
            row.update(status='INCONCLUSIVE' if category in INFRA else 'FAIL', category=category,
                       fingerprint=fingerprint('simulation', 'go', category, suite))
        if row['status'] == 'PASS' and not row['tests']:
            row.update(status='FAIL', category='malformed_output', fingerprint=fingerprint('simulation', 'go', 'malformed_output', suite))
        checks.append(row)
    return checks


def fixture(layer, scenario):
    # Fixed controlled fixtures; never fake a live result. Production refuses them.
    checks = ([result(c, 'go') for c in ('tidy', 'integrity')] + [result('security', 'govulncheck')]
              if layer == 'daily' else [result('cli', 'codex', contract='exec', version=BASELINE['tools']['codex']['version'],
                                             expected={'--model': True}, observed={'--model': True})])
    if scenario in ('drift', 'infrastructure', 'security'):
        category = {'drift': 'contract_drift', 'infrastructure': 'network', 'security': 'vulnerability'}[scenario]
        checks[-1] = result('security' if scenario == 'security' else 'cli',
                            'govulncheck' if scenario == 'security' else 'codex', category,
                            contract='' if scenario == 'security' else 'exec')
    return checks


def manifest(layer, checks, scenario='live'):
    revision = execute(['git', 'rev-parse', 'HEAD'])['out'].strip()
    if not re.fullmatch('[0-9a-f]{40}', revision):
        raise ValueError('subject revision unavailable')
    repository = os.environ.get('GITHUB_REPOSITORY', 'local/axiom')
    if not re.fullmatch(r'[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+', repository):
        raise ValueError('invalid repository')
    run_id = os.environ.get('GITHUB_RUN_ID', '0')
    attempt = os.environ.get('GITHUB_RUN_ATTEMPT', '1')
    if not run_id.isdecimal() or not attempt.isdecimal():
        raise ValueError('invalid run identity')
    status = 'FAIL' if any(c['status'] == 'FAIL' for c in checks) else (
        'INCONCLUSIVE' if any(c['status'] == 'INCONCLUSIVE' for c in checks) else 'PASS')
    branch = 'main' if repository == 'rgomids/axiom' else os.environ.get('GITHUB_REF_NAME', 'main')
    if not re.fullmatch(r'[a-z0-9][a-z0-9/-]{0,100}', branch):
        raise ValueError('invalid branch')
    return {'schema_version': 1, 'repository': repository, 'workflow': 'periodic-monitoring',
            'run_id': run_id, 'run_attempt': attempt,
            'run_url': f'https://github.com/{repository}/actions/runs/{run_id}',
            'timestamp': datetime.now(timezone.utc).isoformat(), 'branch': branch,
            'subject_sha': revision, 'layer': layer, 'scenario': scenario,
            'baseline_digest': BASELINE_DIGEST, 'status': status, 'checks': checks}


def write_evidence(data, output):
    encoded = json.dumps(data, indent=2) + '\n'
    if len(encoded.encode()) > ARTIFACT_LIMIT:
        raise ValueError('evidence exceeds limit')
    output.mkdir(parents=True, exist_ok=True)
    (output / 'manifest.json').write_text(encoded, encoding='utf-8')
    summary = f'Periodic {data["layer"]}: **{data["status"]}** — `{data["subject_sha"]}`\n\n'
    summary += '| Check | Component/contract | Result | Category |\n|---|---|---|---|\n'
    for row in data['checks']:
        summary += f'| {row["check"]} | {row["component"]}/{row["contract"]} | {row["status"]} | {row["category"]} |\n'
    if any(row['check'] == 'security' and row['status'] != 'PASS' for row in data['checks']):
        summary += '\nSecurity: maintainers must rerun privately and follow SECURITY.md. No advisory details are retained.\n'
    (output / 'summary.md').write_text(summary, encoding='utf-8')
    if os.environ.get('GITHUB_OUTPUT'):
        with open(os.environ['GITHUB_OUTPUT'], 'a', encoding='utf-8') as stream:
            stream.write(f'subject={data["subject_sha"]}\n')
    if os.environ.get('GITHUB_STEP_SUMMARY'):
        with open(os.environ['GITHUB_STEP_SUMMARY'], 'a', encoding='utf-8') as stream:
            stream.write(summary)


def main():
    global DEADLINE
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=['run'])
    parser.add_argument('--layer', choices=['daily', 'weekly'], required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--scenario', choices=['live', 'clean', 'drift', 'infrastructure', 'security'], default='live')
    args = parser.parse_args()
    if args.scenario != 'live' and (os.environ.get('GITHUB_REPOSITORY', 'rgomids/axiom') == 'rgomids/axiom'):
        parser.error('controlled scenarios require a sandbox repository')
    if (args.scenario in ('drift', 'infrastructure') and args.layer != 'weekly') or (args.scenario == 'security' and args.layer != 'daily'):
        parser.error('drift/infrastructure fixtures require weekly; security fixture requires daily')
    DEADLINE = time.monotonic() + 14 * 60
    with tempfile.TemporaryDirectory(prefix='axiom-periodic-') as directory:
        home = Path(directory)
        env = clean_environment(home)
        checks = fixture(args.layer, args.scenario) if args.scenario != 'live' else (
            daily(home, env) if args.layer == 'daily' else weekly(home, env))
        data = manifest(args.layer, checks, args.scenario)
        write_evidence(data, args.output)
    print(f'Periodic {args.layer}: {data["status"]}; sanitized evidence written')
    return 0 if data['status'] == 'PASS' else 1


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (OSError, ValueError):
        print('Periodic monitoring: INCONCLUSIVE (runner/evidence failure)', file=sys.stderr)
        sys.exit(1)
