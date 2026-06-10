# ADR 0005 — Custo Hipotético vs. Custo Real

## Status
Aceito (limitação metodológica deliberada e declarada).

## Contexto
O experimento ocorre em cluster local k3s. Não há fatura cloud real. Reportar um valor monetário sem contexto seria enganoso.

## Decisão
Toda saída de `projected_monthly_waste_usd` é declaradamente **hipotética**, ancorada no modelo estático do [ADR 0001](0001-modelo-custo-fargate.md). O valor responde à pergunta:

> *Se esta mesma carga de trabalho rodasse em AWS Fargate (us-east-1, on-demand), quanto custaria o desperdício mensal observado?*

Esta limitação deve aparecer:

1. Nos capítulos de **metodologia** e **limitações** do TCC.
2. Na **descrição do construto** (Catálogo de Indicadores FinOps).
3. (Opcionalmente, futuramente) no payload JSON, via campo `cost_model: "aws-fargate-us-east-1-static"`.

## Alternativas consideradas
- **Custo real do cluster local** (CPU + RAM × tarifa de energia local): muito complexo de auditar e específico do hardware do pesquisador.
- **Não reportar valor financeiro**: rejeitado — descaracteriza o construto FinOps proposto.

## Consequências
- A H1 é validada contra **um modelo de custo, não contra uma fatura**. O critério "erro < 10%" compara `projected_monthly_waste_usd` (reportado) contra `Cost(theoretical)` (calculado manualmente sobre os inputs brutos coletados).
- O endpoint `/api/v1/audit` automatiza essa comparação para reprodução científica.
