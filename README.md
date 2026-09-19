# go-analyze

Serviço de análise FinOps para desperdício de CPU e memória em workloads Kubernetes. Coleta requests, usage e limits do Prometheus, persiste snapshots/consolidações e projeta custos mensais separados por recurso.

## Indicadores e custos

- `cpu_waste_ratio` e `mem_waste_ratio` são limitados a `[0,1]`.
- `cpu_projected_monthly_waste_usd` e `memory_projected_monthly_waste_usd` detalham o custo.
- `projected_monthly_waste_usd` permanece compatível e equivale à soma dos dois custos positivos por workload.
- Limits servem à redistribuição de métricas sem label de container; não geram indicador de risco.

## Execução

Copie `.env.example`, configure PostgreSQL e Prometheus e execute `go run ./cmd`. A especificação OpenAPI gerada fica em `docs/swagger.json` e a interface em `/swagger/`.

O reset `POST /api/v1/experiment/reset` existe somente no binário combinado, fica desabilitado por padrão e exige simultaneamente `EXPERIMENT_RESET_ENABLED=true` e `Authorization: Bearer <EXPERIMENT_RESET_TOKEN>`.

## Notificações

As séries `go_analyze_notifications_total`, `go_analyze_notification_last_timestamp_seconds` e `go_analyze_notification_sustained_duration_seconds` usam os labels `indicator`, `scope`, `namespace`, `pod` e `container`. `scope` é `global` ou `workload`. A latência de sinalização é a duração sustentada observada menos a duração configurada. Um intervalo sem observações maior que duas vezes o intervalo de consolidação reinicia a condição; reiniciar o processo também reinicia o tracker em memória.

## Validação

```bash
go test ./...
go test -race ./...
go vet ./...
cd ../tests/finops-eval
python -m unittest discover -s tests
python -m harness.run --dry-run
```

O harness intercala quatro cenários (CPU/memória, positivo/negativo) com seed registrada e exclui rodadas inválidas dos denominadores. A auditoria declara `audit_type=internal_consistency`; ela testa coerência interna do modelo, não valida uma fatura real.
