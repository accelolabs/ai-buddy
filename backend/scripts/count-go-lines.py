#!/usr/bin/env python3
"""Count nonblank Go lines excluding comment-only lines; helpers count as tests."""
import argparse
from pathlib import Path
import re
import subprocess

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--git-ref', help='Measure a committed revision instead of the worktree')
parser.add_argument('--services-only', action='store_true', help='Exclude the retired template')
args = parser.parse_args()
root = Path(__file__).resolve().parent.parent
repo_root = root.parent
if args.git_ref:
    paths = subprocess.check_output(
        ['git', 'ls-tree', '-r', '--name-only', args.git_ref], cwd=repo_root, text=True
    ).splitlines()
    sources = ((path, subprocess.check_output(
        ['git', 'show', f'{args.git_ref}:{path}'], cwd=repo_root, text=True
    )) for path in paths if path.endswith('.go'))
else:
    sources = ((str(path.relative_to(root)), path.read_text())
               for directory in ('cmd', 'internal')
               for path in (root / directory).rglob('*.go'))
counts = {'application': 0, 'tests': 0}
for path, source in sources:
    if args.services_only and path.startswith('backend-template/'):
        continue
    kind = 'tests' if (path.endswith('_test.go') or '/testdb/' in path
                       or '/testutil/' in path) else 'application'
    source = re.sub(r'/\*.*?\*/', '', source, flags=re.S)
    counts[kind] += sum(bool(line.strip()) and not line.lstrip().startswith('//')
                        for line in source.splitlines())
for kind, count in counts.items():
    print(f'{kind}: {count}')
