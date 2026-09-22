"""
deploy.py — sobe e derruba cenários no cluster, com espera por readiness.
Espelha o 'kubectl apply' + 'kubectl wait' que você fazia à mão.
"""
import time
from . import config
from pathlib import Path

ROOT = config.ROOT


def ensure_namespace(ns: str):
    """Cria o namespace se não existir (idempotente)."""
    existing = config.kubectl("get", "ns", ns, "--ignore-not-found",
                              "-o", "name", check=False)
    if not existing:
        config.kubectl("create", "namespace", ns)


def apply_scenario(manifest_rel: str):
    """kubectl apply do manifesto do cenário."""
    manifest_path = str(ROOT / manifest_rel)
    config.kubectl("apply", "-f", manifest_path)


def delete_scenario(manifest_rel: str):
    """Remove o cenário. Não levanta se já não existir."""
    manifest_path = str(ROOT / manifest_rel)
    config.kubectl("delete", "-f", manifest_path,
                   "--ignore-not-found", "--wait=true", check=False)


def wait_namespace_empty(ns: str, timeout_seconds: int = 60):
    """
    Bloqueia até NÃO haver mais pods de teste no namespace.
    Evita contaminação entre cenários: o ratio global do go-analyze
    reflete todos os pods, então a próxima rodada só pode começar quando
    o pod anterior tiver desaparecido de fato (não só recebido o delete).
    """
    import time
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        out = config.kubectl(
            "get", "pods", "-n", ns,
            "-l", "finops-test=true",
            "-o", "name", check=False,
        )
        if not out.strip():
            return
        time.sleep(2)
    # se chegou aqui, ainda há pods — segue mesmo assim, mas avisa
    raise RuntimeError(f"namespace {ns} ainda tem pods de teste após {timeout_seconds}s")


def wait_pod_ready(ns: str, pod: str, timeout_seconds: int = 120):
    """Bloqueia até o pod ficar Ready, ou levanta após timeout."""
    config.kubectl(
        "wait", "--for=condition=Ready", f"pod/{pod}",
        "-n", ns, f"--timeout={timeout_seconds}s",
    )