# ADR 0004 — Intervalo de Scraping 30s e Consolidação 1m

## Status
Aceito (substitui orientação inicial de 5min descrita no `IMPL.md`).

## Contexto
O `projeto-pesquisa.docx` originalmente menciona "scrape a cada 5 min" e o Prometheus configurado a 15s. Durante a implementação ficou claro que:

- 5 minutos é uma frequência baixa para detectar a latência de sinalização < 40s exigida pelos critérios de aceitação.
- O scrape do próprio go-analyze precisa ser mais denso que o do Prometheus para produzir histórico granular.

## Decisão
- `SCRAPE_INTERVAL = 30s` (configurável).
- `CONSOLIDATE_INTERVAL = 1m`, agregando janelas deslizantes de `CONSOLIDATE_WINDOW = 10m`.

A latência de sinalização é dominada por `SCRAPE_INTERVAL + CONSOLIDATE_INTERVAL + RequiredDuration_sustained`. O período de sustentação (3d/7d) é o termo dominante, não o intervalo de scraping.

## Alternativas consideradas
- **5min original**: rejeitado — viola critério "< 40s" e produz histórico granular demais para análise de tendências.
- **15s = mesmo do Prometheus**: rejeitado — pressão I/O desnecessária no Postgres do experimento.

## Consequências
- Cobertura granular para validação científica.
- ~120 inserts/hora no Postgres em condição normal — irrelevante para volume de experimento.
- A divergência em relação ao `IMPL.md` está documentada aqui; o `projeto-pesquisa.docx` final deve mencionar o ajuste.
