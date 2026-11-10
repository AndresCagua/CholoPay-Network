import unittest
from unittest.mock import patch

from runner import Result, request_once, run


class RunnerTests(unittest.TestCase):
    def test_deterministic_failure_injection_does_not_call_target(self):
        with patch("runner.urllib.request.urlopen") as urlopen:
            result = request_once("http://synthetic.invalid", 1, 2)
        self.assertEqual(result.error, "injected_failure")
        urlopen.assert_not_called()

    def test_run_reports_successes_and_injected_failures(self):
        class Response:
            status = 200
            def __enter__(self): return self
            def __exit__(self, *_): return False
            def read(self): return b'{"status":"UP"}'

        with patch("runner.urllib.request.urlopen", return_value=Response()):
            summary = run("http://synthetic.local/health/live", 5, 2, 2)
        self.assertEqual(summary["requests"], 5)
        self.assertEqual(summary["successful"], 3)
        self.assertEqual(summary["injected_failures"], 2)
        self.assertEqual(summary["failed"], 2)

    def test_invalid_parameters_are_rejected(self):
        with self.assertRaises(ValueError):
            run("http://synthetic.local", 0, 1, 0)
        with self.assertRaises(ValueError):
            run("http://synthetic.local", 1, 1, -1)


if __name__ == "__main__":
    unittest.main()
