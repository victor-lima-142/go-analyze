#!/usr/bin/env python3
"""Run the acceptance command suite and record exit codes and durations."""
import json
import os
import shlex
import subprocess
import time
from pathlib import Path

from . import config

ROOT = config.ROOT
GO = ROOT.parent.parent / "go-analyze"


def execute(name, command, cwd):
    started = time.monotonic()
    proc = subprocess.run(command, cwd=cwd, text=True, capture_output=True, env=os.environ.copy())
    duration = round(time.monotonic() - started, 3)
    log = ROOT / "results" / "logs" / f"validation-{name}.log"
    log.parent.mkdir(parents=True, exist_ok=True)
    log.write_text(proc.stdout + proc.stderr)
    return {"name": name, "command": shlex.join(command), "exit_code": proc.returncode,
            "duration_seconds": duration, "log": str(log.relative_to(ROOT)),
            "summary": (proc.stdout + proc.stderr).strip().splitlines()[-1:]}


def main():
    kubectl = shlex.split(os.environ.get("KUBECTL", "kubectl"))
    commands = [
        ("go-test", ["go", "test", "./..."], GO),
        ("go-test-race", ["go", "test", "-race", "./..."], GO),
        ("go-vet", ["go", "vet", "./..."], GO),
        ("postgres-integration", ["go", "test", "./pkg/database/migrations", "-run", "TestPostgresFreshUpgradePersistenceTransactionAndCascade", "-v"], GO),
        ("python-compile", [str(ROOT / "env/bin/python"), "-m", "compileall", "-q", "harness", "tests"], ROOT),
        ("python-tests", [str(ROOT / "env/bin/python"), "-m", "unittest", "discover", "-s", "tests", "-v"], ROOT),
        ("campaign-dry-run", [str(ROOT / "env/bin/python"), "-m", "harness.run", "--dry-run"], ROOT),
        ("contracts", [str(ROOT / "env/bin/python"), "-m", "harness.contracts"], ROOT),
        ("manifests-server-dry-run", [*kubectl, "apply", "--dry-run=server", "-f", "k8s/postgres.yaml", "-f", "k8s/go-analyze.yaml", "-f", "manifests/"], ROOT),
        ("git-diff-check", ["git", "diff", "--check"], GO),
    ]
    results = [execute(*spec) for spec in commands]
    payload = {"commands": results, "ok": all(r["exit_code"] == 0 for r in results)}
    (ROOT / "results" / "validation.json").write_text(json.dumps(payload, indent=2, ensure_ascii=False) + "\n")
    print(json.dumps({"ok": payload["ok"], "passed": sum(r["exit_code"] == 0 for r in results), "total": len(results)}))
    return 0 if payload["ok"] else 1


if __name__ == "__main__": raise SystemExit(main())
