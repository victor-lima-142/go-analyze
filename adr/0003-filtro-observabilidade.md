# ADR 0003 — Filtro Configurável de Observabilidade

## Status
Aceito.

## Contexto
Pods/containers do próprio stack de monitoramento (Prometheus, kube-state-metrics, node-exporter, alertmanager, etc.) também aparecem nas queries PromQL. Se entrarem no cálculo, contaminam os indicadores de desperdício, inflando o "overprovisioning" do experimento.

## Decisão
Centralizar o filtro em `internal/observability/filter.go`, configurável por variáveis de ambiente:

- `OBSERVABILITY_NAMESPACES` (CSV) — default: `kube-system,kubernetes-dashboard`.
- `OBSERVABILITY_PATTERNS` (CSV) — substrings (case-insensitive) que, se encontradas em `pod` ou `container`, eliminam o sample. Default: `prometheus,kube-state-metrics,node-exporter,pushgateway,configmap-reload,alertmanager`.

Toda lógica anterior `isPrometheusOrSystem` e o filtro duplicado em `sumValues` agora delegam a este componente.

## Alternativas consideradas
- **Manter hardcoded**: rejeitado — limita extensibilidade para clusters com Datadog, Grafana Agent, Loki, etc.
- **Lista de allowlist por namespace**: rejeitado — clusters experimentais têm muitos namespaces dinâmicos.

## Consequências
- Reproduções do experimento em outros clusters podem ajustar o filtro sem mexer no código.
- Testes verificam custom filter + fallback para default quando vazio.
