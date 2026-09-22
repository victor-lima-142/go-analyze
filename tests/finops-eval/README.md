# finops-eval

Harness reproduzível para quatro cenários Kubernetes: desperdício de memória, memória bem dimensionada, desperdício de CPU e CPU próxima do request. A ordem das rodadas é embaralhada pela seed registrada em `experiment.yaml`.

Antes da campanha, configure o serviço combinado com `EXPERIMENT_RESET_ENABLED=true`, defina um `EXPERIMENT_RESET_TOKEN` forte no serviço e exporte o mesmo valor no ambiente do harness. A maturação deve ser estritamente maior que janela + duração sustentada + margem; o dry-run valida essa condição sem tocar no cluster.

```bash
python -m unittest discover -s tests
python -m harness.run --dry-run
python -m harness.run --reps 2
python -m harness.run
```

Durante a maturação, o harness faz polling de notificações usando todos os labels do workload (`indicator`, `scope=workload`, `namespace`, `pod`, `container`), ignorando séries globais e outros pods. Rodadas sem item ou auditoria conclusiva são `invalid`; exceções são `error`. Somente rodadas `valid` entram nos denominadores.

Saídas:

- `results/per_run.csv`: custo CPU/memória/total, valor do indicador, latência, classificação, status e erro;
- `results/summary.csv`: TP/FN/FP/TN, contagens válidas/inválidas/erro, taxas e intervalo Wilson de 95%;
- `results/metadata.json`: commit, configuração efetiva, timestamps, versões e imagens.

A latência é `duração_sustentada_observada - duração_sustentada_configurada`. A auditoria mede consistência interna do cálculo e não representa confronto com faturamento real.
