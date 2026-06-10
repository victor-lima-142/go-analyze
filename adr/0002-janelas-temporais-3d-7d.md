# ADR 0002 — Janelas Temporais de Detecção (3d para CPU/Mem, 7d para PVC)

## Status
Aceito.

## Contexto
A definição operacional de overprovisioning exige que o desperdício seja **sustentado** ao longo de uma janela mínima — caso contrário, picos transitórios gerariam falsos positivos.

## Decisão
- **CPU e Memória**: ratio > 0,50 sustentado por **3 dias consecutivos**.
- **PVC**: ratio > 0,50 sustentado por **7 dias consecutivos**.
- **HPA**: `avg_replicas / max_replicas` agregado em janela de **3 dias** via PromQL (`avg_over_time(...[3d])`).

As janelas são configuráveis via `HPA_WINDOW` e `PVC_WINDOW`.

## Alternativas consideradas
- **Sem janela** (instantâneo): rejeitado — gera falsos positivos em picos.
- **Janela única de 8 dias** (default VPA): rejeitado — atrasaria detecção em CPU/Mem além do necessário.
- **Janelas iguais para PVC e CPU**: rejeitado — volumes têm maior inércia operacional; redimensionar disco custa mais.

## Consequências
- A janela do HPA está hardcoded como `[3d]` em PromQL mas alimentada pela função `HPAAvgReplicasQuery`/`HPAMaxReplicasQuery` (parâmetro `time.Duration`).
- O tracker de notificação sustentada (`internal/notifications/sustained.go`) usa essas durações para decidir quando emitir alerta.
- O tempo de "warm-up" do experimento é, no mínimo, 3 dias antes de emitir o primeiro veredito sobre CPU/Mem.

## Referência
- Vertical Pod Autoscaler (default 8d): https://github.com/kubernetes/autoscaler
- CAST AI Kubernetes Cost Benchmark Report 2024.
