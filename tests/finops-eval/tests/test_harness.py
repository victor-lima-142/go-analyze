import csv
import tempfile
import unittest
from pathlib import Path

from harness import collect, run, stats


class HarnessTests(unittest.TestCase):
    def test_classification_excludes_invalid(self):
        self.assertEqual(stats.classify_run("positive", True), "TP")
        self.assertEqual(stats.classify_run("negative", True), "FP")
        rows = [{"status": "valid", "notified": True}, {"status": "invalid", "notified": True}]
        agg = stats.aggregate_scenario("cpu", "positive", rows)
        self.assertEqual((agg["TP"], agg["valid_runs"], agg["invalid_runs"]), (1, 1, 1))

    def test_wilson_interval(self):
        low, high = stats.wilson95(5, 10)
        self.assertLess(low, 0.5); self.assertGreater(high, 0.5)
        self.assertEqual(stats.wilson95(0, 0), (None, None))

    def test_exact_metric_labels(self):
        text = '\n'.join([
            'go_analyze_notifications_total{indicator="cpu_waste_ratio",scope="global",namespace="",pod="",container=""} 9',
            'go_analyze_notifications_total{indicator="cpu_waste_ratio",scope="workload",namespace="n",pod="p",container="c"} 2',
            'go_analyze_notifications_total{indicator="cpu_waste_ratio",scope="workload",namespace="n",pod="other",container="c"} 7',
        ])
        labels = {"indicator": "cpu_waste_ratio", "scope": "workload", "namespace": "n", "pod": "p", "container": "c"}
        self.assertEqual(collect.parse_metric(text, "go_analyze_notifications_total", labels), 2)

    def test_reproducible_randomization(self):
        scenarios = [{"name": "a"}, {"name": "b"}, {"name": "c"}, {"name": "d"}]
        first = [(x[0]["name"], x[1]) for x in run.randomized_schedule(scenarios, 3, 42)]
        second = [(x[0]["name"], x[1]) for x in run.randomized_schedule(scenarios, 3, 42)]
        self.assertEqual(first, second)
        self.assertNotEqual(first, [(s["name"], r) for r in range(1, 4) for s in scenarios])

    def test_maturation_validation(self):
        cfg = {"experiment": {"window_seconds": 10, "sustained_duration_seconds": 20, "margin_seconds": 5, "maturation_seconds": 35}, "scenarios": [{}]}
        with self.assertRaises(ValueError): run.validate(cfg)

    def test_csv_schema(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "per_run.csv"
            with path.open("w", newline="") as handle:
                writer = csv.DictWriter(handle, fieldnames=run.PER_RUN_FIELDS); writer.writeheader()
            self.assertIn("cpu_cost_usd", path.read_text())
            self.assertIn("detection_latency_seconds", path.read_text())

    def test_indicator_content_contract(self):
        original = collect.config.http_get_json
        collect.config.http_get_json = lambda _: {"content": [{"namespace": "n", "pod": "p", "container": "c"}]}
        try:
            self.assertEqual(collect.read_indicator("http://x", "n", "p", "c")["pod"], "p")
        finally:
            collect.config.http_get_json = original

    def test_resume_reads_incremental_csv(self):
        original = run.RESULTS
        with tempfile.TemporaryDirectory() as directory:
            run.RESULTS = Path(directory)
            with (run.RESULTS / "per_run.csv").open("w", newline="") as handle:
                writer = csv.DictWriter(handle, fieldnames=run.PER_RUN_FIELDS); writer.writeheader()
                writer.writerow({"scenario": "cpu_positive", "run": 1, "status": "valid"})
            self.assertEqual(run.load_runs()[0]["scenario"], "cpu_positive")
        run.RESULTS = original

    def test_validate_accepts_campaign_timing(self):
        cfg = {"experiment": {"window_seconds": 300, "sustained_duration_seconds": 300, "margin_seconds": 90, "maturation_seconds": 720}, "scenarios": [{}]}
        run.validate(cfg)


if __name__ == "__main__":
    unittest.main()
