import contextlib
import io
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import measure_coverage as coverage


class CoverageTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        (self.root / "backend").mkdir()
        (self.root / "frontend" / "coverage").mkdir(parents=True)
        root = patch.object(coverage, "ROOT", self.root)
        root.start()
        self.addCleanup(root.stop)

    def test_backend_uses_race_fresh_tests_and_standard_go_reports(self):
        output = "backend/file.go:1: Function 50.0%\ntotal:\t(statements)\t95.9%\n"
        with patch.object(coverage, "run", side_effect=[None, output, None]) as run:
            self.assertEqual(coverage.backend(), {"statements": 95.9})
        command = run.call_args_list[0].args[0]
        self.assertIn("-race", command)
        self.assertIn("-count=1", command)
        self.assertIn("-coverpkg=./...", command)
        self.assertEqual((self.root / "backend/coverage/functions.txt").read_text(), output)
        self.assertIn("-html=", run.call_args_list[-1].args[0][3])

    def test_backend_rejects_a_missing_total(self):
        with patch.object(coverage, "run", side_effect=[None, "no source files"]):
            with self.assertRaises(ValueError):
                coverage.backend()

    def test_frontend_reads_each_metric(self):
        names = ("lines", "statements", "functions", "branches")
        summary = {"total": {name: {"pct": 91} for name in names}}
        report = self.root / "frontend/coverage/coverage-summary.json"
        report.write_text(json.dumps(summary))
        with patch.object(coverage, "run") as run:
            self.assertEqual(coverage.frontend(), dict.fromkeys(names, 91))
        run.assert_called_once_with(["npm", "run", "test:coverage"], self.root / "frontend")

    def test_frontend_does_not_read_old_reports_after_test_failure(self):
        with patch.object(coverage, "run", side_effect=subprocess.CalledProcessError(1, "npm")):
            with self.assertRaises(subprocess.CalledProcessError):
                coverage.frontend()

    def test_run_propagates_a_failed_command(self):
        with patch.object(
            coverage.subprocess, "run", side_effect=subprocess.CalledProcessError(1, "go")
        ) as command:
            with self.assertRaises(subprocess.CalledProcessError):
                coverage.run(["go", "test"], self.root)
        command.assert_called_once_with(
            ["go", "test"], cwd=self.root, check=True, text=True, capture_output=False
        )

    def test_cli_checks_both_targets_and_each_metric(self):
        for percent, expected in ((80, 0), (79.99, 1)):
            with (
                self.subTest(percent=percent),
                patch.object(coverage, "backend", return_value={"statements": 95}),
                patch.object(coverage, "frontend", return_value={"lines": 99, "branches": percent}),
                contextlib.redirect_stdout(io.StringIO()),
            ):
                self.assertEqual(coverage.main([]), expected)

    def test_cli_keeps_failure_even_when_another_target_passes(self):
        with (
            patch.object(coverage, "backend", side_effect=FileNotFoundError("go")),
            patch.object(coverage, "frontend", return_value={"lines": 99}) as frontend,
            contextlib.redirect_stdout(io.StringIO()),
            contextlib.redirect_stderr(io.StringIO()),
        ):
            self.assertEqual(coverage.main([]), 1)
            frontend.assert_called_once()

    def test_cli_runs_only_the_requested_target(self):
        with (
            patch.object(coverage, "backend", return_value={"statements": 90}) as backend,
            patch.object(coverage, "frontend") as frontend,
            contextlib.redirect_stdout(io.StringIO()),
        ):
            self.assertEqual(coverage.main(["backend"]), 0)
            backend.assert_called_once()
            frontend.assert_not_called()

    def test_cli_rejects_unreadable_frontend_summary(self):
        with (
            patch.object(coverage, "run"),
            contextlib.redirect_stdout(io.StringIO()),
            contextlib.redirect_stderr(io.StringIO()),
        ):
            self.assertEqual(coverage.main(["frontend"]), 1)


if __name__ == "__main__":
    unittest.main()
