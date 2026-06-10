# ADR 0009 — Recálculo de Ratios sobre Totais no Consolidador

## Status
Aceito.

## Contexto
A implementação inicial do Consolidator calculava a média aritmética simples dos ratios já-calculados (`cpu_waste_ratio`, `mem_waste_ratio`, etc.) entre os snapshots da janela. Isso introduz **viés de Simpson**: workloads com inputs muito diferentes acabam ponderados de forma equivocada.

Exemplo: dois snapshots, ambos com CPU req=1, mas um com used=0.1 e outro com used=0.9. Ratios individuais: 0.9 e 0.1, média = 0.5. Recalculado sobre totais: (2 - 1.0)/2 = 0.5. Convergem nesse caso simétrico — mas se um snapshot tem CPU req=10 e outro req=0.1, a média de ratios distorce.

## Decisão
O Consolidator recalcula **todos os ratios** a partir das somas dos inputs antes de persistir a consolidação:

```go
avgCPUWaste = (sumCPUReq - sumCPUUsed) / sumCPUReq
avgMemWaste = (sumMemReq - sumMemUsed) / sumMemReq
avgPVCWaste = (sumPVCCap - sumPVCUsed) / sumPVCCap
avgHPAEff   = sumHPAAvg / sumHPAMax
```

Os agregados por workload (em `consolidated_workload_snapshots`) seguem a mesma lógica.

## Alternativas consideradas
- **Média aritmética simples**: rejeitado (viés explicado acima).
- **Mediana**: descartada — mais resistente a outliers mas perde a aditividade exigida para o `projected_monthly_waste_usd`.

## Consequências
- Os ratios consolidados são consistentes com os reportados ad-hoc pelo `/api/v1/indicators`.
- A documentação técnica do TCC deve usar a fórmula recalculada como padrão.
