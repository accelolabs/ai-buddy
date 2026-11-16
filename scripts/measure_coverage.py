#!/usr/bin/env python3
"""Run Go/Vitest coverage and enforce an 80% minimum. Standard library only."""

import argparse
import json
from pathlib import Path
import re
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
MINIMUM = 80


def run(command, directory, capture=False):
    print(f"\n> {' '.join(map(str, command))}", flush=True)
    return subprocess.run(
        command, cwd=directory, check=True, text=True, capture_output=capture
    ).stdout


def backend():
    directory = ROOT / "backend"
    report = directory / "coverage"
    report.mkdir(exist_ok=True)
    profile = report / "coverage.out"
    run(
        ["go", "test", "-race", "-count=1", "-coverpkg=./...", f"-coverprofile={profile}", "./..."],
        directory,
    )
    output = run(["go", "tool", "cover", f"-func={profile}"], directory, capture=True)
    (report / "functions.txt").write_text(output, encoding="utf-8")
    match = re.search(r"^total:\s+\(statements\)\s+(\d+(?:\.\d+)?)%$", output, re.MULTILINE)
    if not match:
        raise ValueError("Go did not report total coverage")
    run(["go", "tool", "cover", f"-html={profile}", f"-o={report / 'index.html'}"], directory)
    return {"statements": float(match[1])}


def frontend():
    directory = ROOT / "frontend"
    run(["npm", "run", "test:coverage"], directory)
    report = directory / "coverage" / "coverage-summary.json"
    total = json.loads(report.read_text(encoding="utf-8"))["total"]
    return {name: float(total[name]["pct"]) for name in ("lines", "statements", "functions", "branches")}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("target", nargs="?", choices=("all", "backend", "frontend"), default="all")
    args = parser.parse_args(argv)
    failed = False
    for name, measure in (("backend", backend), ("frontend", frontend)):
        if args.target not in ("all", name):
            continue
        try:
            for metric, percent in measure().items():
                passed = percent >= MINIMUM
                print(f"{name}: {metric} {percent:.2f}% — {'OK' if passed else 'FAIL'} (min {MINIMUM}%)", flush=True)
                failed = failed or not passed
        except (OSError, subprocess.CalledProcessError, ValueError, KeyError) as error:
            print(f"{name}: ERROR: {error}", file=sys.stderr)
            failed = True
    return int(failed)


if __name__ == "__main__":
    sys.exit(main())
