# ADR 0006 — Detecção Sustentada e Sistema de Notificações

## Status
Aceito.

## Contexto
Os critérios de aceitação do TCC exigem mensurar:
- **Taxa de detecção** (>= 90%).
- **Latência de sinalização** (< 40s a partir do término da janela sustentada).
- **Taxa de falsos positivos** (< 5%).

Para isso, é necessário emitir um evento discreto sempre que um workload atinge a condição de overprovisioning sustentado conforme [ADR 0002](0002-janelas-temporais-3d-7d.md).

## Decisão
- Componente `internal/notifications/Tracker` mantém o estado `(IndicatorKey → started_above_at)` em memória.
- Para cada `Observe(key, value)`:
  1. Se `value` cruza o threshold para cima, marca o início do período.
  2. Se permanece acima e ultrapassa `RequiredDuration`, dispara `Notifier.Notify(...)`.
  3. Cooldown configurável evita notificações repetidas para o mesmo workload.
  4. Se `value` cai abaixo, o estado reseta.
- Implementação default do `Notifier` é `SlogNotifier` (log estruturado + counter Prometheus).
- O Consolidator é o ponto natural de observação: ele já agrega janelas deslizantes e é onde a decisão "ratio sustentado" é tomada.

## Alternativas consideradas
- **Persistir estado no Postgres**: rejeitado — adiciona complexidade ao MVP do TCC; o estado em memória é suficiente porque o tracker é alimentado a cada consolidação e perda de estado em restart só atrasa a primeira notificação.
- **Webhooks/Slack/Email diretos**: fora do escopo do MBA; pode ser plugado como novo `Notifier` no futuro.

## Consequências
- O endpoint `/metrics` expõe `go_analyze_notifications_total{indicator="..."}` para medição quantitativa da taxa de detecção.
- O tracker é testável (`sustained_test.go`) com clock injetável.
