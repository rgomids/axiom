#!/usr/bin/env python3
"""Import a reviewed Axiom DS landing bundle without a build-time network dependency."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import shutil
import subprocess

ROOT = Path(__file__).resolve().parents[1]
MANIFEST = '.axiom-landing-source.json'
LEGACY_FILES = {'index.html', 'styles.css', 'lang.js', 'diagram.js', 'skills.js'}


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def safe_files(directory):
    if directory.is_symlink() or not directory.is_dir():
        raise ValueError('bundle must be a real directory')
    files = {}
    for path in sorted(directory.rglob('*')):
        if path.is_symlink() or not path.resolve().is_relative_to(directory.resolve()):
            raise ValueError('bundle contains a symlink or unsafe path')
        if path.is_file():
            name = path.relative_to(directory).as_posix()
            if name == MANIFEST or any(part.startswith('.') and part != '.nojekyll' for part in path.relative_to(directory).parts):
                raise ValueError('bundle contains a reserved or hidden path')
            files[name] = path
    for required in ('index.html', 'app.js', 'storytelling.js', 'story-state.js', 'tokens.css'):
        if required not in files:
            raise ValueError('bundle missing ' + required)
    return files


def sync(source, revision, target=ROOT / 'site'):
    source = source.resolve()
    if not re.fullmatch(r'[0-9a-f]{40}', revision):
        raise ValueError('revision must be a full lowercase Git SHA')
    checkout = subprocess.check_output(['git', '-C', str(source), 'rev-parse', '--show-toplevel'], text=True).strip()
    if source != (Path(checkout) / 'packages/landing/dist').resolve():
        raise ValueError('bundle must be the canonical packages/landing/dist in its checkout')
    actual = subprocess.check_output(['git', '-C', checkout, 'rev-parse', 'HEAD'], text=True).strip()
    if actual != revision:
        raise ValueError('bundle checkout HEAD does not match the reviewed revision')
    if subprocess.check_output(['git', '-C', checkout, 'status', '--porcelain', '--', 'packages/landing', 'packages/brand', 'packages/tokens', 'scripts/build-landing.mjs', 'scripts/tokens.mjs'], text=True).strip():
        raise ValueError('canonical landing sources have uncommitted changes')
    files = safe_files(source)
    target = target.resolve()
    if target.is_symlink() or target == source or target.is_relative_to(source):
        raise ValueError('unsafe target directory')
    previous = target / MANIFEST
    owned = set(json.loads(previous.read_text(encoding='utf-8'))['files']) if previous.exists() else LEGACY_FILES
    for relative in owned:
        path = target / relative
        if not path.resolve().is_relative_to(target) or path.is_symlink():
            raise ValueError('unsafe existing manifest path')
    for relative, path in files.items():
        destination = target / relative
        if destination.is_symlink() or not destination.resolve().is_relative_to(target):
            raise ValueError('unsafe destination path')
        if destination.exists() and relative not in owned:
            raise ValueError('refusing to overwrite an unowned file: ' + relative)
    for relative in owned - files.keys():
        path = target / relative
        if path.is_file():
            path.unlink()
    target.mkdir(parents=True, exist_ok=True)
    for relative, path in files.items():
        destination = target / relative
        destination.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(path, destination)
    manifest = {'schema_version': 1, 'repository': 'https://github.com/rgomids/axiom-ds',
                'revision': revision, 'bundle': 'packages/landing/dist',
                'files': {name: digest(path) for name, path in files.items()}}
    previous.write_text(json.dumps(manifest, indent=2) + '\n', encoding='utf-8')
    print('PASS: imported reviewed landing bundle at ' + revision)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--bundle', required=True, type=Path)
    parser.add_argument('--revision', required=True)
    args = parser.parse_args()
    try:
        sync(args.bundle, args.revision)
    except (ValueError, OSError, subprocess.CalledProcessError, KeyError) as error:
        parser.exit(1, 'FAIL: ' + str(error) + '\n')
