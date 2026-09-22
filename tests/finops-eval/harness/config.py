"""
config.py — carrega experiment.yaml e expõe helpers de baixo nível
para falar com o go-analyze, o Prometheus e o kubectl.

Nenhuma lógica de experimento aqui; só leitura de config e chamadas cruas.
"""
import os
import shlex
import subprocess
import urllib.request
import json
import yaml
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


def load_config(path: str = None) -> dict:
    cfg_path = Path(path) if path else (ROOT / "experiment.yaml")
    with open(cfg_path) as f:
        return yaml.safe_load(f)


def http_get_json(url: str, timeout: int = 30) -> dict:
    """GET que devolve JSON parseado. Levanta em erro de rede/HTTP."""
    req = urllib.request.Request(url, method="GET")
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return json.loads(resp.read().decode())


def http_get_text(url: str, timeout: int = 30) -> str:
    """GET que devolve texto cru (usado para o /metrics do Prometheus)."""
    req = urllib.request.Request(url, method="GET")
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return resp.read().decode()


def http_post(url: str, token: str, timeout: int = 30) -> dict:
    """POST sem corpo (usado para o /api/v1/experiment/reset)."""
    req = urllib.request.Request(url, method="POST", data=b"", headers={"Authorization": f"Bearer {token}"})
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        body = resp.read().decode()
        try:
            return json.loads(body)
        except json.JSONDecodeError:
            return {"raw": body}


def kubectl(*args: str, check: bool = True) -> str:
    """Executa kubectl e devolve stdout. Levanta em erro se check=True."""
    cmd = [*shlex.split(os.environ.get("KUBECTL", "kubectl")), *args]
    proc = subprocess.run(cmd, capture_output=True, text=True)
    if check and proc.returncode != 0:
        raise RuntimeError(
            f"kubectl {' '.join(args)} falhou (rc={proc.returncode}): {proc.stderr.strip()}"
        )
    return proc.stdout.strip()
