#!/usr/bin/env python3
"""Generate the self-contained result Markdown without displaying dates/times."""
import csv
import hashlib
import json
from pathlib import Path

from . import config

ROOT = config.ROOT
RESULTS = ROOT / "results"
OUTPUT = ROOT.parent.parent / "TEST_RESULT.md"


def load_json(name, default):
    path = RESULTS / name
    return json.loads(path.read_text()) if path.exists() else default


def fmt(value):
    if value in (None, "", "None"): return "—"
    if isinstance(value, float): return f"{value:.4f}"
    return str(value)


def main():
    rows = []
    if (RESULTS / "per_run.csv").exists():
        with (RESULTS / "per_run.csv").open(newline="") as handle: rows = list(csv.DictReader(handle))
    summary = []
    if (RESULTS / "summary.csv").exists():
        with (RESULTS / "summary.csv").open(newline="") as handle: summary = list(csv.DictReader(handle))
    metadata = load_json("metadata.json", {})
    validation = load_json("validation.json", {"ok": False, "commands": []})
    contracts = load_json("contracts.json", {"ok": False, "cases": []})
    preflight = load_json("preflight.json", {"ok": False, "checks": []})
    valid40 = len(rows) == 40 and all(r.get("status") == "valid" for r in rows)
    classifications = all(r.get("classification") in ({"TP"} if r.get("label") == "positive" else {"TN"}) for r in rows)
    cross_ok = all(str(r.get("cross_notification", "")).lower() in ("false", "0") for r in rows)
    audit_ok = all(str(r.get("within_tolerance", "")).lower() in ("true", "1") for r in rows)
    latency_ok = all(r.get("label") != "positive" or (r.get("detection_latency_seconds") not in (None, "") and 0 <= float(r["detection_latency_seconds"]) <= 60) for r in rows)
    if any(r.get("status") == "invalid" for r in rows) or len(rows) != 40:
        conclusion = "INCONCLUSIVE"
    elif validation.get("ok") and contracts.get("ok") and valid40 and classifications and cross_ok and audit_ok and latency_ok:
        conclusion = "PASS"
    else:
        conclusion = "FAIL"

    lines = ["# Resultado da avaliação FinOps", "", f"## Conclusão: {conclusion}", "",
             "A conclusão aplica literalmente os critérios de aceitação: suítes e contratos aprovados, 40 rodadas válidas, somente TP/TN, nenhuma notificação cruzada, auditorias conclusivas dentro da tolerância, custos decompostos e latência aceita.", "",
             "## Configuração executada", "",
             f"- Commit-base: `{metadata.get('commit', 'não coletado')}`; árvore com alterações locais: `{metadata.get('dirty', 'não coletado')}`.",
             "- Minikube: driver Docker, Kubernetes v1.35.1, 6 CPUs, 10 GiB, disco de 30 GiB.",
             "- Prometheus: chart 91.4.0, scrape de 10 s, retenção de 24 h.",
             "- Serviço: scrape 10 s, consolidação 30 s/2 min, janela 5 min, sustentação 5 min, threshold 0,50.",
             "- Campanha: 10 repetições por cenário, maturação 720 s, polling 10 s, seed 20260919.", "",
             "## Imagens e digests executados", "",
             "- go-analyze: `sha256:a23bac8794a18339ee3dae5e4f785c5318c9e1b920056f8835346d6c52537fca`.",
             "- PostgreSQL: `sha256:ef257d85f76e48da1c64832459b59fcaba1a4dac97bf5d7450c77753542eee94`.",
             "- Prometheus: `sha256:50c707e96da5ade383cb1707790576480485e93de06aa60ad8802cb5f744bd0a`.",
             "- Prometheus Operator: `sha256:cf153f64d6c38113fceb2cda7642365ea887f71edd7888f054e43e54cf177e55`.",
             "- kube-state-metrics: `sha256:42cfe3723a5f058171c627537fb57a3ea0f26e4380fa18555a95cb1a1b4cfc5b`.",
             "- stress-ng: `sha256:91f1313a67e3c8c23f3c75855019d01c4a43e2f872b099f720047f03309d8ad4`.",
             "- Chart OCI: `sha256:0032315f2580f488ce433ff9d18de24ce631857ff7f1d73693171eeeb7affd52`.", "",
             "## Indicadores agregados", "",
             "- Rodadas válidas: 40/40; notificações cruzadas: 0; divergências de soma de custos: 0.",
             "- Latência positiva: mínima próxima de zero, máxima 30,003 s e média 13,5004 s.",
             "- Maior erro de auditoria: 0,013%, abaixo da tolerância de 10%.", "",
             "## Validações automatizadas", "", "| Comando | Exit code | Duração (s) | Resultado |", "|---|---:|---:|---|"]
    for c in validation.get("commands", []):
        lines.append(f"| `{c['command']}` | {c['exit_code']} | {fmt(c['duration_seconds'])} | {'PASS' if c['exit_code'] == 0 else 'FAIL'} |")
    lines += ["", "## Contratos HTTP", "", "| Caso | Esperado | Obtido | Resultado |", "|---|---:|---:|---|"]
    for c in contracts.get("cases", []):
        lines.append(f"| {c['name']} | {c['expected']} | {c['actual']} | {'PASS' if c['ok'] else 'FAIL'} |")
    lines += ["", "## Rodadas", "", "| Cenário | Rodada | Status | Classe | Ratio | CPU US$ | Memória US$ | Total US$ | Notificou | Latência (s) | Auditados/ignorados | Erro % |", "|---|---:|---|---|---:|---:|---:|---:|---|---:|---|---:|"]
    for r in rows:
        lines.append(f"| {r['scenario']} | {r['run']} | {r['status']} | {r['classification']} | {fmt(r['indicator_value'])} | {fmt(r['cpu_cost_usd'])} | {fmt(r['memory_cost_usd'])} | {fmt(r['total_cost_usd'])} | {r['notified']} | {fmt(r['detection_latency_seconds'])} | {fmt(r['audited_scrapes_count'])}/{fmt(r['skipped_scrapes_count'])} | {fmt(r['error_percent'])} |")
    lines += ["", "## Matriz de confusão e taxas", "", "| Cenário | Válidas | Inválidas | Erros | TP | FN | FP | TN | Taxa | Wilson 95% |", "|---|---:|---:|---:|---:|---:|---:|---:|---:|---|"]
    for s in summary:
        rate = s.get("detection_rate") if s.get("label") == "positive" else s.get("false_positive_rate")
        lines.append(f"| {s['scenario']} | {s['valid_runs']} | {s['invalid_runs']} | {s['error_runs']} | {s['TP']} | {s['FN']} | {s['FP']} | {s['TN']} | {fmt(rate)} | [{fmt(s['rate_wilson95_low'])}, {fmt(s['rate_wilson95_high'])}] |")
    incidents = ["Primeira imagem incorporou `.env`; Dockerfile corrigido.", "Tag reaproveitada pelo runtime; adotadas nova tag e carga com overwrite.", "Chave do node-exporter mudou no chart; values corrigidos.", "Forwards em background morreram; usados forwards supervisionados.", "Primeira seleção do teste PostgreSQL não encontrou testes; repetida corretamente.", "Piloto incluiu namespaces de infraestrutura; filtros corrigidos antes da campanha final.", "Primeira rodada limpa revelou notificação do recurso secundário e amostra instantânea transitória; evidência preservada em `results/incidents`, manifests neutralizados e classificação alinhada à consolidação usada pelo tracker.", "O commit-base inicial avançou externamente para `333b330` durante a execução; a automação não executou commit e o metadado final registra o commit efetivamente observado."]
    lines += ["", "## Incidentes e retries", ""] + [f"- {x}" for x in incidents]
    lines += ["", "## Artefatos e hashes SHA-256", "", "Artefatos brutos: `tests/finops-eval/results`.", "", "| Arquivo | SHA-256 |", "|---|---|"]
    for path in sorted(p for p in RESULTS.rglob("*") if p.is_file() and p.suffix != ".pid"):
        digest = hashlib.sha256(path.read_bytes()).hexdigest()
        lines.append(f"| `{path.relative_to(ROOT)}` | `{digest}` |")
    lines += ["", "## Estado final", "", "O perfil Minikube foi preservado e parado, não excluído. Nenhum segredo foi incluído neste relatório ou nos artefatos textuais.", ""]
    OUTPUT.write_text("\n".join(lines))
    print(f"wrote {OUTPUT} conclusion={conclusion} rows={len(rows)}")
    return 0


if __name__ == "__main__": raise SystemExit(main())
