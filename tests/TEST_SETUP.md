# Ambiente e procedimento da avaliação FinOps

## Escopo e estado inicial

A inspeção ocorreu na raiz `/home/victor-reboredo/Documents/mba`. O código Go está em `go-analyze/`; harness, manifests e automação estão em `tests/finops-eval/`. O repositório Go estava no commit-base `68f8225` (`main`) e já possuía alterações não commitadas, que foram preservadas. A raiz do MBA não é um único repositório Git.

A máquina tinha 12 CPUs lógicas, 31 GiB de RAM, 8 GiB de swap e 255 GiB livres. Ferramentas observadas: Minikube 1.38.1, Docker Engine 29.8.1, Helm 4.3.0, kubectl host 1.37.0, Go 1.27.1 e Python 3.14.7. O cliente canônico é `minikube -p mba-finops kubectl --`, compatível com o servidor 1.35.1. Não havia perfil `mba-finops`.

A porta 5432 estava ocupada por um PostgreSQL Docker. Após autorização expressa para limpar todo o Docker, container, imagens, rede, caches e volume foram removidos. As portas experimentais 8081, 9090 e 55432 ficaram livres. Os resultados anteriores foram movidos, de forma recuperável, para `/tmp/finops-eval-results-before-20260919`.

O baseline aprovou 75 testes Go em 18 pacotes e os 6 testes Python então existentes. Após ampliar o harness, 9 testes Python foram aprovados. A configuração do harness já usava 300 s de sustentação, mas um `.env` local ignorado pelo Git continha valores divergentes; sua inclusão acidental na primeira imagem provocou o incidente descrito adiante. Durante a execução, as alterações Go pré-existentes foram commitadas fora do harness como `333b330`; nenhum comando `git commit` foi executado por esta automação. Por isso `68f8225` é o estado inicial e `333b330` é o commit efetivamente registrado ao fim da campanha.

## Arquitetura efetiva

```text
harness Python
  ├─ localhost:8081 ─ port-forward ─ Service/go-analyze ─ Deployment/go-analyze
  │                                      ├─ HTTP → Prometheus
  │                                      └─ SQL → PostgreSQL
  ├─ localhost:9090 ─ port-forward ─ Prometheus
  ├─ localhost:55432 ─ port-forward ─ PostgreSQL
  └─ minikube kubectl ─ create/delete ─ Pods de carga
```

Os namespaces são `monitoring`, `finops-system` e `finops-test`. O primeiro contém Prometheus Operator, Prometheus e kube-state-metrics; Grafana, Alertmanager, regras padrão e node-exporter estão desabilitados. O segundo contém PostgreSQL e go-analyze, seus Services, ConfigMap e Secret. O terceiro contém no máximo um pod experimental por vez.

Kube-state-metrics fornece requests e limits das queries de CPU e memória. Kubelet/cAdvisor fornece uso de CPU (`container_cpu_usage_seconds_total`) e memória (`container_memory_working_set_bytes`). Prometheus coleta a cada 10 s e retém 24 h. O serviço coleta a cada 10 s, consolida a cada 30 s sobre 2 min, analisa uma janela de 5 min e exige 5 min sustentados acima de 0,50.

O Minikube usa driver Docker, Kubernetes v1.35.1, 6 CPUs, 10 GiB e disco de 30 GiB. O PostgreSQL existe somente no cluster e é exposto em 55432. Segredos não são gravados em logs ou Markdown: o setup gera token aleatório se necessário, guarda-o somente no Secret e o wrapper o lê em memória.

Versões fixadas: `kube-prometheus-stack:91.4.0`, digest OCI `sha256:0032315f2580f488ce433ff9d18de24ce631857ff7f1d73693171eeeb7affd52`; carga `alexeiled/stress-ng@sha256:91f1313a67e3c8c23f3c75855019d01c4a43e2f872b099f720047f03309d8ad4`; aplicação local `go-analyze:finops-eval-20260919-v2`. O digest efetivamente executado é registrado no resultado.

## Modelo experimental

Cada cenário tem 10 repetições:

- `memory_positive`: request 512 MiB, limit 640 MiB e carga aproximada de 150 MiB; sem request de CPU para neutralizar o indicador não testado. Espera ratio de memória `> 0,50`, custo de memória positivo e notificação.
- `memory_negative`: request 256 MiB, limit 320 MiB e carga aproximada de 240 MiB; sem request de CPU. Espera ratio de memória `<= 0,50` e nenhuma notificação.
- `cpu_positive`: request 1 core, limit 1,5 core e carga aproximada de 10%; memória com request 8 MiB e limit 64 MiB. Espera ratio de CPU `> 0,50`, custo de CPU positivo e notificação.
- `cpu_negative`: request 900m, limit 2 cores e carga aproximada de 90%; memória com request 8 MiB e limit 64 MiB. Espera ratio de CPU `<= 0,50` e nenhuma notificação.

Para request `R` e uso `U`, `ratio = clamp((R-U)/R, 0, 1)`. Custo mensal de CPU é `max(R-U,0) × 0,04048 × 720`; memória é `max(R-U,0)/GiB × 0,004445 × 720`; total é a soma arredondada a quatro casas. É auditoria de consistência interna do modelo estático, não confronto com faturamento real.

A janela e a condição sustentada têm 300 s; margem de 90 s; maturação de 720 s. Latência é a duração sustentada observada menos 300 s e deve ficar entre zero e 60 s. Positivo notificado é TP; positivo não notificado, FN; negativo notificado, FP; negativo não notificado, TN. `valid` exige workload exato, ratio esperado, soma de custos com tolerância 0,0001, auditoria conclusiva abaixo de 10%, scrape, consolidação, detalhes atuais e ausência de notificação cruzada. Evidência incompleta gera `invalid`; exceção operacional gera `error`. Inválidas tornam o resultado inconclusivo. Os intervalos de 95% usam Wilson.

A execução é sequencial porque cada rodada trunca tabelas, reseta trackers e depende de isolamento. A ordem é embaralhada deterministicamente com seed `20260919`. O CSV é persistido atomicamente após cada rodada; a retomada usa `(scenario, run)` e repete erros.

## Procedimento realmente executado

1. Inspeção com `minikube version`, `docker version`, `helm version --short`, `kubectl version`, `go version`, `python --version`, `nproc`, `free -h`, `df -h` e `ss`.
2. Limpeza autorizada com `docker system prune -af --volumes`, remoção explícita do container e volume restantes. O banco antigo não é recuperável; os resultados antigos foram apenas movidos.
3. Criação com `minikube start -p mba-finops --driver=docker --kubernetes-version=v1.35.1 --cpus=6 --memory=10240 --disk-size=30g`.
4. Instalação do chart OCI com Helm. A primeira revisão deixou node-exporter ativo porque a versão usa `nodeExporter.enabled`; a chave foi corrigida e a revisão seguinte confirmou somente Operator, kube-state-metrics e Prometheus.
5. Criação do Secret em memória, aplicação de `k8s/postgres.yaml` e confirmação por rollout/`pg_isready`.
6. Build, carga e aplicação do go-analyze. A primeira imagem incorporou `.env`, sobrescreveu o ConfigMap e tentou `localhost:5432`. Removeu-se `COPY .env*` do Dockerfile. O runtime reutilizou o digest da tag anterior; uma tag nova e `image load --overwrite=true` corrigiram o cache. O rollout passou e `/healthz` reportou banco e Prometheus saudáveis.
7. Port-forwards em background foram encerrados pelo executor mesmo com PID/nohup. A execução efetiva mantém uma sessão supervisionada com os três forwards.
8. O preflight validou contexto, namespace, health, métricas internas, readiness do Prometheus, séries kube-state/cAdvisor e reset autenticado.
9. Foram executados `go test ./...`, `go test -race ./...`, `go vet ./...`, integração PostgreSQL real, compilação Python, testes unitários, dry-run das 40 rodadas, validação server-side e `git diff --check`. A primeira seleção do teste PostgreSQL não encontrou testes; a repetição com o nome correto aprovou fresh/upgrade.
10. A primeira matriz HTTP ocorreu com forwards mortos e falhou por conexão recusada. Com a sessão supervisionada, 29/29 contratos e as invariantes semânticas passaram.
11. A primeira rodada-piloto mostrou três workloads nos agregados: aplicação e banco não estavam excluídos. Ela foi interrompida antes de persistir resultado; `monitoring` e `finops-system` foram adicionados a `OBSERVABILITY_NAMESPACES`, o rollout foi reiniciado e o preflight repetido. A rodada limpa seguinte foi preservada como inválida: CPU negativa gerou notificação de memória porque o recurso secundário também estava superdimensionado, e a leitura instantânea coincidiu com uma amostra transitória zero. Os manifests passaram a neutralizar o recurso secundário e a classificação passou a usar a consolidação mais recente, mantendo o instantâneo como evidência. A campanha final executa, por rodada: reset, confirmação de vazio, settle, baseline, apply, Ready, espera da série, polling, coleta, classificação, JSON de evidência e limpeza em `finally`.
12. Ao fim são coletados objetos, targets, logs e digests; gera-se o resultado e executa-se `bin/teardown.sh`, que para o perfil sem excluí-lo.

## Reprodução e operação

```bash
cd /home/victor-reboredo/Documents/mba/tests/finops-eval
# opcional: export EXPERIMENT_RESET_TOKEN="$(openssl rand -hex 32)"
bin/setup.sh

# terminal dedicado para os forwards
minikube -p mba-finops kubectl -- -n finops-system port-forward svc/go-analyze 8081:8080 &
minikube -p mba-finops kubectl -- -n monitoring port-forward svc/monitoring-kube-prometheus-prometheus 9090:9090 &
minikube -p mba-finops kubectl -- -n finops-system port-forward svc/postgres 55432:5432 & wait

# outro terminal
bin/run-with-cluster-token.sh env/bin/python -m harness.preflight
bin/run-with-cluster-token.sh env/bin/python -m harness.contracts
bin/run-with-cluster-token.sh env/bin/python -m harness.run
```

Os valores não secretos ficam em `experiment.yaml` e nos manifests. Um token próprio deve ser exportado apenas no ambiente antes do setup. Acompanhe com `tail -f results/logs/campaign.log`, `column -s, -t results/per_run.csv` e `kubectl -n finops-test get pods -w`. `Ctrl-C` é seguro: `finally` remove o pod e a retomada ignora rodadas já persistidas. Logs: `minikube -p mba-finops kubectl -- -n finops-system logs deployment/go-analyze`. Targets: `curl -s localhost:9090/api/v1/targets`.

Para regenerar relatórios, execute `env/bin/python -m harness.report`. Para parar: `bin/teardown.sh`; para reiniciar: `minikube start -p mba-finops` e refaça os forwards; para excluir definitivamente: `minikube delete -p mba-finops` — não executado nesta avaliação.

Troubleshooting: em falha de pull, valide DNS/proxy e repita o setup; target `DOWN`, consulte `/api/v1/targets`, ServiceMonitor e endpoints; pod sem métricas, confirme Ready, aguarde dois scrapes e consulte as famílias necessárias; forward encerrado, reinicie a sessão e repita preflight; banco indisponível, verifique `pg_isready`, logs e endpoints; rodada interrompida é limpa e retomada pelo mesmo comando. Nunca copie o Secret para logs.
