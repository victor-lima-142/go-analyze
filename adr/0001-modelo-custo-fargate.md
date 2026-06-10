# ADR 0001 — Modelo de Custo de Referência (AWS Fargate)

## Status
Aceito.

## Contexto
A pesquisa precisa converter métricas de subutilização (vCPU/RAM ociosas) em uma projeção financeira para validar a hipótese H1 com erro inferior a 10%. O experimento ocorre em cluster local k3s, onde não há fatura real.

## Decisão
Adotar o modelo de custo on-demand do **AWS Fargate (região us-east-1)**:

```
Cost_hypothetical = [(CPU_waste_cores × 0.04048) + (RAM_waste_GB × 0.004445)] × 720
```

Os valores 0.04048 USD/vCPU-hora e 0.004445 USD/GB-hora são parametrizáveis via `CPU_HOURLY_USD` e `MEMORY_GIB_HOURLY_USD`. O multiplicador 720 (horas/mês) é configurável via `MONTHLY_HOURS`.

## Alternativas consideradas
- **AWS EC2 on-demand**: fatura instâncias inteiras, dificultando a separação CPU/memória.
- **GKE/AKS**: estruturas semelhantes ao Fargate, porém menos documentadas em literatura cloud-native.
- **Spot/Reserved**: voláteis; quebram a auditabilidade.

## Consequências
- O custo reportado é hipotético — ver [ADR 0005](0005-custo-hipotetico-vs-real.md).
- Mudanças na tabela de preços da AWS exigem atualização do `.env` (sem recompilar).
- O artefato é agnóstico ao provedor; trocar de modelo significa apenas parametrizar outras constantes.
