#!/usr/bin/env python3
"""Executable HTTP contract matrix with machine-readable evidence."""
import argparse
import json
import os
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

from . import config


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def request(base, method, path, token=None, follow=True):
    headers = {"Authorization": f"Bearer {token}"} if token is not None else {}
    req = urllib.request.Request(base + path, data=b"" if method == "POST" else None, headers=headers, method=method)
    opener = urllib.request.build_opener() if follow else urllib.request.build_opener(NoRedirect)
    try:
        with opener.open(req, timeout=45) as response:
            body = response.read().decode(errors="replace")
            return response.status, body, dict(response.headers)
    except urllib.error.HTTPError as exc:
        return exc.code, exc.read().decode(errors="replace"), dict(exc.headers)


def run_contracts(cfg, token):
    base = cfg["experiment"]["analyzer_base_url"]
    cases = [
        ("health", "GET", "/healthz", 200, None, True),
        ("metrics", "GET", "/metrics", 200, None, True),
        ("swagger_redirect", "GET", "/swagger", 301, None, False),
        ("swagger_ui", "GET", "/swagger/index.html", 200, None, True),
        ("swagger_openapi", "GET", "/swagger/doc.json", 200, None, True),
        ("indicators_default", "GET", "/api/v1/indicators", 200, None, True),
        ("indicators_exact", "GET", "/api/v1/indicators?namespace=finops-test&pod=cpu-positive&container=stress&window=5m", 200, None, True),
        ("indicators_prefix", "GET", "/api/v1/indicators?namespace=finops-test&pod=cpu-%25&container=stress", 200, None, True),
        ("indicators_page", "GET", "/api/v1/indicators?page=1&pageSize=1", 200, None, True),
        ("indicators_method", "POST", "/api/v1/indicators", 405, None, True),
        ("consolidated_default", "GET", "/api/v1/consolidated", 200, None, True),
        ("consolidated_no_items", "GET", "/api/v1/consolidated?include_items=false&period=30min", 200, None, True),
        ("consolidated_page", "GET", "/api/v1/consolidated?page=1&pageSize=1", 200, None, True),
        ("consolidated_bad_period", "GET", "/api/v1/consolidated?period=bogus", 400, None, True),
        ("consolidated_bad_date", "GET", "/api/v1/consolidated?start_at=nope", 400, None, True),
        ("consolidated_method", "POST", "/api/v1/consolidated", 405, None, True),
        ("workload_required", "GET", "/api/v1/workloads/details", 400, None, True),
        ("workload_exact", "GET", "/api/v1/workloads/details?namespace=finops-test&pod=cpu-positive&container=stress&limit=2", 200, None, True),
        ("workload_prefix", "GET", "/api/v1/workloads/details?namespace=finops-test&pod=cpu-&container=stress&period=30min", 200, None, True),
        ("workload_bad_date", "GET", "/api/v1/workloads/details?namespace=finops-test&container=stress&start_at=nope", 400, None, True),
        ("workload_method", "POST", "/api/v1/workloads/details", 405, None, True),
        ("audit_period", "GET", "/api/v1/audit?period=30min", 200, None, True),
        ("audit_details", "GET", "/api/v1/audit?period=30min&details=true", 200, None, True),
        ("audit_bad_period", "GET", "/api/v1/audit?period=bogus", 400, None, True),
        ("audit_method", "POST", "/api/v1/audit", 405, None, True),
        ("reset_missing", "POST", "/api/v1/experiment/reset", 401, None, True),
        ("reset_invalid", "POST", "/api/v1/experiment/reset", 401, "invalid-token", True),
        ("reset_method", "GET", "/api/v1/experiment/reset", 405, token, True),
        ("reset_valid", "POST", "/api/v1/experiment/reset", 200, token, True),
    ]
    results = []
    for name, method, path, expected, auth, follow in cases:
        status, body, headers = request(base, method, path, auth, follow)
        results.append({"name": name, "method": method, "path": path, "expected": expected,
                        "actual": status, "ok": status == expected, "content_type": headers.get("Content-Type", ""),
                        "body_excerpt": body[:300]})
    status, body, _ = request(base, "GET", "/api/v1/indicators?pageSize=1000")
    semantic = {"ratios_in_range": False, "cost_sum": False}
    if status == 200:
        payload = json.loads(body)
        items = payload.get("content", [])
        semantic["ratios_in_range"] = all(0 <= i.get("cpu_waste_ratio", -1) <= 1 and 0 <= i.get("mem_waste_ratio", -1) <= 1 for i in items)
        semantic["cost_sum"] = all(abs(i.get("cpu_projected_monthly_waste_usd", 0) + i.get("memory_projected_monthly_waste_usd", 0) - i.get("projected_monthly_waste_usd", 0)) <= 0.0001 for i in items)
    return {"timestamp": datetime.now(timezone.utc).isoformat(), "cases": results, "semantic": semantic,
            "ok": all(r["ok"] for r in results) and all(semantic.values())}


def main(argv=None):
    parser = argparse.ArgumentParser(); parser.add_argument("--config"); parser.add_argument("--output", default=str(config.ROOT / "results" / "contracts.json"))
    args = parser.parse_args(argv); cfg = config.load_config(args.config)
    token = os.environ.get(cfg["experiment"]["reset_token_env"], "")
    if not token: raise RuntimeError("missing reset token")
    result = run_contracts(cfg, token)
    path = Path(args.output); path.parent.mkdir(parents=True, exist_ok=True); path.write_text(json.dumps(result, indent=2, ensure_ascii=False) + "\n")
    print(json.dumps({"ok": result["ok"], "passed": sum(c["ok"] for c in result["cases"]), "total": len(result["cases"]), "semantic": result["semantic"]}))
    return 0 if result["ok"] else 1


if __name__ == "__main__": raise SystemExit(main())
