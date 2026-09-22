#!/usr/bin/env python3
"""Fail-fast validation of every dependency needed by a campaign."""
import argparse
import json
import os
from datetime import datetime, timezone
from pathlib import Path

from . import collect, config, run


def check(cfg, token=None):
    exp = cfg["experiment"]
    run.validate(cfg)
    base = exp["analyzer_base_url"]
    prom = exp["prometheus_base_url"]
    checks = []

    def record(name, fn):
        try:
            detail = fn()
            checks.append({"name": name, "ok": True, "detail": detail})
        except Exception as exc:
            checks.append({"name": name, "ok": False, "detail": str(exc)})

    record("kubectl_context", lambda: config.kubectl("config", "current-context"))
    record("namespace", lambda: config.kubectl("get", "namespace", exp["namespace"], "-o", "name"))
    record("healthz", lambda: config.http_get_json(f"{base}/healthz"))
    record("metrics", lambda: "go_analyze_scrapes_total" in config.http_get_text(f"{base}/metrics"))
    record("prometheus_ready", lambda: config.http_get_text(f"{prom}/-/ready").strip())
    for metric in ("kube_pod_container_resource_requests", "container_cpu_usage_seconds_total", "container_memory_working_set_bytes"):
        record(f"metric:{metric}", lambda metric=metric: len(collect.prometheus_query(prom, f"count({metric})")) > 0)
    if token:
        record("reset_authenticated", lambda: config.http_post(f"{base}/api/v1/experiment/reset", token).get("status") == "ok")
    return {"timestamp": datetime.now(timezone.utc).isoformat(), "checks": checks,
            "ok": all(c["ok"] and c["detail"] is not False for c in checks)}


def main(argv=None):
    parser = argparse.ArgumentParser()
    parser.add_argument("--config")
    parser.add_argument("--output", default=str(config.ROOT / "results" / "preflight.json"))
    args = parser.parse_args(argv)
    cfg = config.load_config(args.config)
    token = os.environ.get(cfg["experiment"]["reset_token_env"])
    result = check(cfg, token)
    path = Path(args.output); path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(result, indent=2, ensure_ascii=False) + "\n")
    print(json.dumps(result, indent=2, ensure_ascii=False))
    return 0 if result["ok"] else 1


if __name__ == "__main__":
    raise SystemExit(main())
