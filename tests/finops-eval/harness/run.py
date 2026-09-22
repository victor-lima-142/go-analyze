#!/usr/bin/env python3
"""Reproducible, resumable four-scenario Kubernetes experiment runner."""
import argparse
import csv
import json
import os
import platform
import random
import subprocess
import time
from datetime import datetime, timezone

from . import collect, config, deploy, reset, stats

ROOT = config.ROOT
RESULTS = ROOT / "results"
EVIDENCE = RESULTS / "evidence"
PER_RUN_FIELDS = [
    "scenario", "label", "indicator", "run", "status", "classification", "notified",
    "cross_notification", "indicator_value", "cpu_cost_usd", "memory_cost_usd", "total_cost_usd",
    "detected_at", "detection_latency_seconds", "scrapes_count", "consolidations_count",
    "audited_scrapes_count", "skipped_scrapes_count", "audit_reported_usd", "audit_recalculated_usd",
    "error_percent", "within_tolerance", "started_at", "timestamp", "error",
]
SUMMARY_FIELDS = ["scenario", "label", "runs", "valid_runs", "invalid_runs", "error_runs",
                  "TP", "FN", "FP", "TN", "detection_rate", "false_positive_rate",
                  "rate_wilson95_low", "rate_wilson95_high"]


def log(message):
    print(f"[{datetime.now().strftime('%H:%M:%S')}] {message}", flush=True)


def validate(cfg):
    exp = cfg["experiment"]
    minimum = exp["window_seconds"] + exp["sustained_duration_seconds"] + exp["margin_seconds"]
    if exp["maturation_seconds"] <= minimum:
        raise ValueError(f"maturation_seconds must be greater than window+sustained+margin ({minimum})")
    if exp["sustained_duration_seconds"] != 300:
        raise ValueError("sustained_duration_seconds must be 300 for this campaign")
    if not cfg.get("scenarios"):
        raise ValueError("at least one scenario is required")


def randomized_schedule(scenarios, repetitions, seed):
    schedule = [(scenario, run) for run in range(1, repetitions + 1) for scenario in scenarios]
    random.Random(seed).shuffle(schedule)
    return schedule


def notification_snapshot(metrics_url, metric):
    text = config.http_get_text(metrics_url)
    result = {}
    import re
    for line in text.splitlines():
        if not line.startswith(metric + "{"):
            continue
        match = re.match(r'[^\{]+\{([^}]*)\}\s+([0-9eE.+-]+)$', line)
        if not match:
            continue
        labels = dict(re.findall(r'(\w+)="((?:\\.|[^"])*)"', match.group(1)))
        key = tuple(labels.get(k, "") for k in ("indicator", "scope", "namespace", "pod", "container"))
        result[key] = float(match.group(2))
    return result


def wait_for_series(exp, scenario, timeout=180):
    query = (f'kube_pod_container_resource_requests{{namespace="{exp["namespace"]}",'
             f'pod="{scenario["pod"]}",container="{scenario["container"]}"}}')
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        if collect.prometheus_query(exp["prometheus_base_url"], query):
            return
        time.sleep(5)
    raise TimeoutError(f"Prometheus series not available after {timeout}s: {scenario['pod']}")


def run_one(exp, scenario, run_idx, token):
    base, ns = exp["analyzer_base_url"], exp["namespace"]
    metrics_url = f"{base}/metrics"
    labels = (scenario["indicator"], ns, scenario["pod"], scenario["container"])
    started_at = datetime.now(timezone.utc).isoformat()
    deployed = False
    try:
        reset.reset_experiment(base, token)
        deploy.wait_namespace_empty(ns)
        time.sleep(exp["settle_seconds"])
        before_all = notification_snapshot(metrics_url, exp["notification_metric"])
        before = collect.read_notification_counter(metrics_url, exp["notification_metric"], *labels)
        deploy.apply_scenario(scenario["manifest"]); deployed = True
        deploy.wait_pod_ready(ns, scenario["pod"])
        wait_for_series(exp, scenario)

        started = time.monotonic(); detected_at = None
        deadline = started + exp["maturation_seconds"]
        while time.monotonic() < deadline:
            now = collect.read_notification_counter(metrics_url, exp["notification_metric"], *labels)
            if now > before and detected_at is None:
                detected_at = datetime.now(timezone.utc).isoformat()
            time.sleep(min(exp["polling_seconds"], max(0, deadline - time.monotonic())))

        live_item = collect.read_indicator(base, ns, scenario["pod"], scenario["container"], exp["window"])
        audit = collect.audit_cost(base)
        consolidated = collect.read_consolidated(base)
        consolidated_item = None
        for consolidation in consolidated.get("content", []):
            for candidate in consolidation.get("items", []) or []:
                if (candidate.get("namespace"), candidate.get("pod"), candidate.get("container")) == (ns, scenario["pod"], scenario["container"]):
                    consolidated_item = candidate
                    break
            if consolidated_item is not None:
                break
        item = consolidated_item or live_item
        details = collect.read_workload_details(base, ns, scenario["pod"], scenario["container"], exp["window"])
        after_all = notification_snapshot(metrics_url, exp["notification_metric"])
        after = collect.read_notification_counter(metrics_url, exp["notification_metric"], *labels)
        observed = collect.read_sustained_duration(metrics_url, *labels)
        notified = after > before
        expected_key = (scenario["indicator"], "workload", ns, scenario["pod"], scenario["container"])
        cross = any(value > before_all.get(key, 0) for key, value in after_all.items()
                    if key[1] == "workload" and key != expected_key)
        ratio = item.get(scenario["indicator"]) if item else None
        cpu_cost = item.get("cpu_projected_monthly_waste_usd") if item else None
        memory_cost = item.get("memory_projected_monthly_waste_usd") if item else None
        total_cost = item.get("projected_monthly_waste_usd") if item else None
        sum_ok = item is not None and abs((cpu_cost + memory_cost) - total_cost) <= 0.0001
        ratio_ok = ratio is not None and ((ratio > 0.5) == (scenario["label"] == "positive"))
        cost_ok = scenario["label"] != "positive" or ((memory_cost if scenario["indicator"].startswith("mem") else cpu_cost) > 0)
        consolidations_count = consolidated.get("totalItems", 0)
        valid = all((item is not None, audit.get("conclusive") is True, audit.get("within_10pct_tolerance") is True,
                     audit.get("audited_scrapes_count", 0) > 0, consolidations_count > 0, sum_ok, ratio_ok, cost_ok,
                     not cross, details.get("current") is not None))
        status = "valid" if valid else "invalid"
        rec = {
            "scenario": scenario["name"], "label": scenario["label"], "indicator": scenario["indicator"],
            "run": run_idx, "status": status, "notified": notified, "cross_notification": cross,
            "indicator_value": ratio, "cpu_cost_usd": cpu_cost, "memory_cost_usd": memory_cost,
            "total_cost_usd": total_cost, "detected_at": detected_at,
            "detection_latency_seconds": max(0, observed - exp["sustained_duration_seconds"]) if notified else None,
            "scrapes_count": audit.get("scrapes_count"), "consolidations_count": consolidations_count,
            "audited_scrapes_count": audit.get("audited_scrapes_count"),
            "skipped_scrapes_count": audit.get("skipped_scrapes_count"),
            "audit_reported_usd": audit.get("reported_projected_monthly_waste_usd"),
            "audit_recalculated_usd": audit.get("recalculated_projected_monthly_waste_usd"),
            "error_percent": audit.get("error_percent"), "within_tolerance": audit.get("within_10pct_tolerance"),
            "started_at": started_at, "timestamp": datetime.now(timezone.utc).isoformat(), "error": "",
        }
        rec["classification"] = stats.classify_run(scenario["label"], notified, status)
        evidence = {"record": rec, "indicator_item": live_item, "classification_item": item, "audit": audit, "consolidated": consolidated,
                    "workload_details": details, "notifications_before": {"|".join(k): v for k, v in before_all.items()},
                    "notifications_after": {"|".join(k): v for k, v in after_all.items()}}
        EVIDENCE.mkdir(parents=True, exist_ok=True)
        (EVIDENCE / f"{scenario['name']}-{run_idx:02d}.json").write_text(json.dumps(evidence, indent=2, ensure_ascii=False) + "\n")
        return rec
    finally:
        if deployed:
            deploy.delete_scenario(scenario["manifest"])
        deploy.wait_namespace_empty(ns)


def version(command):
    try:
        proc = subprocess.run(command, capture_output=True, text=True, timeout=30)
        value = (proc.stdout or proc.stderr).strip().splitlines()
        return value[0] if value else f"exit {proc.returncode}"
    except Exception as exc:
        return f"unavailable: {exc}"


def load_runs():
    path = RESULTS / "per_run.csv"
    if not path.exists():
        return []
    with path.open(newline="") as handle:
        return list(csv.DictReader(handle))


def write_outputs(cfg, runs, started_at, finished_at):
    RESULTS.mkdir(exist_ok=True)
    tmp = RESULTS / "per_run.csv.tmp"
    with tmp.open("w", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=PER_RUN_FIELDS, extrasaction="ignore"); writer.writeheader(); writer.writerows(runs)
    tmp.replace(RESULTS / "per_run.csv")
    summary = [stats.aggregate_scenario(sc["name"], sc["label"], [r for r in runs if r["scenario"] == sc["name"]]) for sc in cfg["scenarios"]]
    with (RESULTS / "summary.csv").open("w", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=SUMMARY_FIELDS); writer.writeheader(); writer.writerows(summary)
    metadata = {"commit": version(["git", "-C", str(ROOT.parent.parent / "go-analyze"), "rev-parse", "HEAD"]),
                "dirty": bool(version(["git", "-C", str(ROOT.parent.parent / "go-analyze"), "status", "--porcelain"])),
                "configuration": cfg, "started_at": started_at, "finished_at": finished_at,
                "versions": {"python": platform.python_version(), "go": version(["go", "version"]),
                             "kubectl": version([*os.environ.get("KUBECTL", "kubectl").split(), "version", "--client"]),
                             "cluster": version([*os.environ.get("KUBECTL", "kubectl").split(), "version"])}}
    (RESULTS / "metadata.json").write_text(json.dumps(metadata, ensure_ascii=False, indent=2) + "\n")


def main(argv=None):
    parser = argparse.ArgumentParser()
    parser.add_argument("--config"); parser.add_argument("--reps", type=int)
    parser.add_argument("--dry-run", action="store_true"); parser.add_argument("--no-resume", action="store_true")
    args = parser.parse_args(argv); cfg = config.load_config(args.config); validate(cfg)
    exp = cfg["experiment"]; reps = args.reps or exp["repetitions"]
    schedule = randomized_schedule(cfg["scenarios"], reps, exp["seed"])
    if args.dry_run:
        log(f"DRY RUN seed={exp['seed']} rounds={len(schedule)}")
        for scenario, run_idx in schedule: log(f"{run_idx}: {scenario['name']} ({scenario['label']})")
        return 0
    token = os.environ.get(exp["reset_token_env"], "")
    if not token: raise RuntimeError(f"missing reset token env {exp['reset_token_env']}")
    deploy.ensure_namespace(exp["namespace"])
    runs = [] if args.no_resume else load_runs()
    completed = {(r["scenario"], int(r["run"])) for r in runs if r.get("status") in ("valid", "invalid")}
    started = datetime.now(timezone.utc).isoformat()
    for scenario, idx in schedule:
        if (scenario["name"], idx) in completed:
            log(f"SKIP completed {scenario['name']} run={idx}"); continue
        log(f"START {scenario['name']} run={idx}")
        try:
            rec = run_one(exp, scenario, idx, token)
        except (KeyboardInterrupt, SystemExit):
            write_outputs(cfg, runs, started, datetime.now(timezone.utc).isoformat()); raise
        except Exception as exc:
            rec = {key: None for key in PER_RUN_FIELDS}
            rec.update(scenario=scenario["name"], label=scenario["label"], indicator=scenario["indicator"],
                       run=idx, status="error", classification="error", notified=False,
                       started_at=datetime.now(timezone.utc).isoformat(), timestamp=datetime.now(timezone.utc).isoformat(), error=str(exc))
        runs = [r for r in runs if not (r["scenario"] == scenario["name"] and int(r["run"]) == idx)] + [rec]
        write_outputs(cfg, runs, started, datetime.now(timezone.utc).isoformat())
        log(f"END {scenario['name']} run={idx} status={rec['status']} class={rec['classification']}")
    write_outputs(cfg, runs, started, datetime.now(timezone.utc).isoformat()); return 0


if __name__ == "__main__":
    raise SystemExit(main())
