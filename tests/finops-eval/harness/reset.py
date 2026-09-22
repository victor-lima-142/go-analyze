"""
reset.py — zera o estado do go-analyze entre rodadas.
Chama o endpoint POST /api/v1/experiment/reset que você implementou,
o qual trunca as tabelas (scrapes, consolidations CASCADE) e reseta
os trackers de notificação sustentada.
"""
from . import config


def reset_experiment(base_url: str, token: str):
    url = f"{base_url}/api/v1/experiment/reset"
    resp = config.http_post(url, token)
    status = resp.get("status")
    if status != "ok":
        raise RuntimeError(f"reset não retornou status ok: {resp}")
    return resp
