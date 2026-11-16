#!/usr/bin/env python3
"""Check statement coverage from go test -coverpkg=./... -coverprofile=PATH ./... ."""
import argparse
import collections
import json
import pathlib
import re
import subprocess
import sys

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('profile', type=pathlib.Path)
parser.add_argument('--minimum', type=float, default=80)
args = parser.parse_args()
package_json = subprocess.check_output(['go', 'list', '-json', './...'], text=True)
packages = []
decoder = json.JSONDecoder()
while package_json.strip():
    package, end = decoder.raw_decode(package_json.lstrip())
    packages.append(package)
    package_json = package_json.lstrip()[end:]

blocks = {}
for line in args.profile.read_text().splitlines()[1:]:
    location, statements, count = line.rsplit(' ', 2)
    size, hits = int(statements), int(count)
    if location in blocks:
        old_size, old_hits = blocks[location]
        if old_size != size:
            raise ValueError(f'Conflicting coverage block: {location}')
        hits = max(hits, old_hits)
    blocks[location] = (size, hits)
totals = collections.defaultdict(lambda: [0, 0])
for location, (size, hits) in blocks.items():
    package = location.rsplit('/', 1)[0]
    totals[package][1] += size
    if hits:
        totals[package][0] += size
# Profiles made without -coverpkg can omit production files. Do not silently
# report an omitted command as a package containing only types.
profile_files = {location.rsplit(':', 1)[0] for location in blocks}
failed = False
for package in packages:
    for name in package.get('GoFiles', []) + package.get('CgoFiles', []):
        source = (pathlib.Path(package['Dir']) / name).read_text()
        if re.search(r'(?m)^func\s', source):
            expected = package['ImportPath'] + '/' + name
            if expected not in profile_files:
                print(f'Missing production file in profile: {expected}', file=sys.stderr)
                failed = True
for package in [p['ImportPath'] for p in packages] + ['total']:
    covered, total = (map(sum, zip(*totals.values())) if package == 'total' else totals[package])
    if not total:
        print(f'{package:45} no measurable statements')
        continue
    percent = 100 * covered / total
    print(f'{package:45} {percent:6.2f}% ({covered}/{total})')
    failed |= percent < args.minimum
if not blocks or not sum(v[1] for v in totals.values()):
    sys.exit('Empty coverage profile')
sys.exit(1 if failed else 0)
