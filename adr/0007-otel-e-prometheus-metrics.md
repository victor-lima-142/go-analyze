# ADR 0007 — Instrumentação OpenTelemetry + Métricas Prometheus

## Status
Aceito.

## Contexto
O `projeto-pesquisa.docx` lista a instrumentação com OpenTelemetry como requisito técnico e compatibilidade com Prometheus/Grafana.

## Decisão
- **Tracing**: `internal/telemetry/Init` configura globalmente um `TracerProvider` da OTel SDK. Por default usa `noop` (zero custo); quando `OTEL_ENABLED=true`, exporta para stdout com sampling 10%. Spans são criados pelos handlers e pelo Calculator.
- **Métricas**: usa `prometheus/client_golang` (já presente como dependência transitiva) com `promauto`. Counters/histograms expostos em `/metrics` via `promhttp.Handler`.
- **Logs estruturados**: stdlib `log/slog` com `TextHandler`.

## Alternativas consideradas
- **OTel Metrics SDK + exporter Prometheus separado**: rejeitado — adiciona dependências sem ganho prático para o MBA. O `client_golang` é suficiente.
- **OTel Collector + Jaeger**: fora do escopo do TCC; o noop default permite a evolução futura sem refactor.

## Consequências
- A aplicação é instrumentada conforme o requisito do TCC mesmo com tracer noop ativo por default.
- Para reprodução com tracing visível, basta `OTEL_ENABLED=true`.
- O `/metrics` permite que o Prometheus do experimento scrape o próprio go-analyze.
