# Resultado da avaliação FinOps

## Conclusão: PASS

A conclusão aplica literalmente os critérios de aceitação: suítes e contratos aprovados, 40 rodadas válidas, somente TP/TN, nenhuma notificação cruzada, auditorias conclusivas dentro da tolerância, custos decompostos e latência aceita.

## Configuração executada

- Commit-base: `333b3300b8224b0bd7f05f04a8628790a4290ab6`; árvore com alterações locais: `True`.
- Minikube: driver Docker, Kubernetes v1.35.1, 6 CPUs, 10 GiB, disco de 30 GiB.
- Prometheus: chart 91.4.0, scrape de 10 s, retenção de 24 h.
- Serviço: scrape 10 s, consolidação 30 s/2 min, janela 5 min, sustentação 5 min, threshold 0,50.
- Campanha: 10 repetições por cenário, maturação 720 s, polling 10 s, seed 20260919.

## Imagens e digests executados

- go-analyze: `sha256:a23bac8794a18339ee3dae5e4f785c5318c9e1b920056f8835346d6c52537fca`.
- PostgreSQL: `sha256:ef257d85f76e48da1c64832459b59fcaba1a4dac97bf5d7450c77753542eee94`.
- Prometheus: `sha256:50c707e96da5ade383cb1707790576480485e93de06aa60ad8802cb5f744bd0a`.
- Prometheus Operator: `sha256:cf153f64d6c38113fceb2cda7642365ea887f71edd7888f054e43e54cf177e55`.
- kube-state-metrics: `sha256:42cfe3723a5f058171c627537fb57a3ea0f26e4380fa18555a95cb1a1b4cfc5b`.
- stress-ng: `sha256:91f1313a67e3c8c23f3c75855019d01c4a43e2f872b099f720047f03309d8ad4`.
- Chart OCI: `sha256:0032315f2580f488ce433ff9d18de24ce631857ff7f1d73693171eeeb7affd52`.

## Indicadores agregados

- Rodadas válidas: 40/40; notificações cruzadas: 0; divergências de soma de custos: 0.
- Latência positiva: mínima próxima de zero, máxima 30,003 s e média 13,5004 s.
- Maior erro de auditoria: 0,013%, abaixo da tolerância de 10%.

## Validações automatizadas

| Comando | Exit code | Duração (s) | Resultado |
|---|---:|---:|---|
| `go test ./...` | 0 | 0.6570 | PASS |
| `go test -race ./...` | 0 | 1.9300 | PASS |
| `go vet ./...` | 0 | 0.1850 | PASS |
| `go test ./pkg/database/migrations -run TestPostgresFreshUpgradePersistenceTransactionAndCascade -v` | 0 | 0.0980 | PASS |
| `/home/victor-reboredo/Documents/mba/tests/finops-eval/env/bin/python -m compileall -q harness tests` | 0 | 0.0390 | PASS |
| `/home/victor-reboredo/Documents/mba/tests/finops-eval/env/bin/python -m unittest discover -s tests -v` | 0 | 0.0770 | PASS |
| `/home/victor-reboredo/Documents/mba/tests/finops-eval/env/bin/python -m harness.run --dry-run` | 0 | 0.0710 | PASS |
| `/home/victor-reboredo/Documents/mba/tests/finops-eval/env/bin/python -m harness.contracts` | 0 | 0.2680 | PASS |
| `minikube -p mba-finops kubectl -- apply --dry-run=server -f k8s/postgres.yaml -f k8s/go-analyze.yaml -f manifests/` | 0 | 0.3740 | PASS |
| `git diff --check` | 0 | 0.0020 | PASS |

## Contratos HTTP

| Caso | Esperado | Obtido | Resultado |
|---|---:|---:|---|
| health | 200 | 200 | PASS |
| metrics | 200 | 200 | PASS |
| swagger_redirect | 301 | 301 | PASS |
| swagger_ui | 200 | 200 | PASS |
| swagger_openapi | 200 | 200 | PASS |
| indicators_default | 200 | 200 | PASS |
| indicators_exact | 200 | 200 | PASS |
| indicators_prefix | 200 | 200 | PASS |
| indicators_page | 200 | 200 | PASS |
| indicators_method | 405 | 405 | PASS |
| consolidated_default | 200 | 200 | PASS |
| consolidated_no_items | 200 | 200 | PASS |
| consolidated_page | 200 | 200 | PASS |
| consolidated_bad_period | 400 | 400 | PASS |
| consolidated_bad_date | 400 | 400 | PASS |
| consolidated_method | 405 | 405 | PASS |
| workload_required | 400 | 400 | PASS |
| workload_exact | 200 | 200 | PASS |
| workload_prefix | 200 | 200 | PASS |
| workload_bad_date | 400 | 400 | PASS |
| workload_method | 405 | 405 | PASS |
| audit_period | 200 | 200 | PASS |
| audit_details | 200 | 200 | PASS |
| audit_bad_period | 400 | 400 | PASS |
| audit_method | 405 | 405 | PASS |
| reset_missing | 401 | 401 | PASS |
| reset_invalid | 401 | 401 | PASS |
| reset_method | 405 | 405 | PASS |
| reset_valid | 200 | 200 | PASS |

## Rodadas

| Cenário | Rodada | Status | Classe | Ratio | CPU US$ | Memória US$ | Total US$ | Notificou | Latência (s) | Auditados/ignorados | Erro % |
|---|---:|---|---|---:|---:|---:|---:|---|---:|---|---:|
| cpu_negative | 6 | valid | TN | 0.0191 | 0.5012 | 0.0052 | 0.5064 | False | — | 76/0 | 0.0002 |
| cpu_negative | 8 | valid | TN | 0.0151 | 0.3963 | 0.0052 | 0.4015 | False | — | 76/0 | 0.0025 |
| memory_negative | 9 | valid | TN | 0 | 0.2436 | 0 | 0.2436 | False | — | 76/0 | 0.0027 |
| memory_positive | 9 | valid | TP | 0.6631 | 0.2713 | 1.061 | 1.3323 | True | 0.0005353120000108902 | 76/0 | 0.0031 |
| memory_negative | 3 | valid | TN | 0 | 0.2857 | 0 | 0.2857 | False | — | 76/0 | 0.0052 |
| memory_positive | 5 | valid | TP | 0.663 | 0.2062 | 1.0609 | 1.2671 | True | 3.144400000110181e-05 | 76/0 | 0.0027 |
| cpu_negative | 10 | valid | TN | 0.0166 | 0.4366 | 0.0052 | 0.4418 | False | — | 76/0 | 0.0004 |
| cpu_negative | 9 | valid | TN | 0.0159 | 0.418 | 0.0051 | 0.4231 | False | — | 76/0 | 0.0074 |
| cpu_positive | 7 | valid | TP | 0.9011 | 26.262 | 0.0052 | 26.2672 | True | 29.99841699199999 | 76/0 | 0 |
| memory_negative | 2 | valid | TN | 0 | 0.0961 | 0 | 0.0961 | False | — | 76/0 | 0.0044 |
| cpu_negative | 3 | valid | TN | 0.0176 | 0.4608 | 0.0051 | 0.4659 | False | — | 76/0 | 0 |
| cpu_positive | 8 | valid | TP | 0.9012 | 26.266 | 0.0051 | 26.2711 | True | 29.99666488600002 | 76/0 | 0.0009 |
| cpu_positive | 4 | valid | TP | 0.9015 | 26.2734 | 0.0051 | 26.2785 | True | 0.0019753390000118998 | 76/0 | 0.0006 |
| memory_positive | 6 | valid | TP | 0.663 | 0.2137 | 1.0609 | 1.2746 | True | 0.0018077359999892906 | 76/0 | 0.0011 |
| memory_negative | 4 | valid | TN | 0 | 0.2779 | 0 | 0.2779 | False | — | 76/0 | 0.0075 |
| memory_negative | 10 | valid | TN | 0 | 0.2363 | 0 | 0.2363 | False | — | 76/0 | 0.013 |
| cpu_positive | 10 | valid | TP | 0.9017 | 26.2795 | 0.0052 | 26.2847 | True | 29.998606216999974 | 76/0 | 0.0005 |
| memory_negative | 8 | valid | TN | 0 | 0.1479 | 0 | 0.1479 | False | — | 76/0 | 0.0003 |
| cpu_negative | 2 | valid | TN | 0.0125 | 0.3278 | 0.0052 | 0.333 | False | — | 76/0 | 0.0015 |
| cpu_negative | 5 | valid | TN | 0.0158 | 0.4152 | 0.0052 | 0.4204 | False | — | 76/0 | 0.002 |
| cpu_negative | 4 | valid | TN | 0.0192 | 0.5027 | 0.0051 | 0.5078 | False | — | 76/0 | 0.0108 |
| memory_negative | 1 | valid | TN | 0 | 0.1875 | 0 | 0.1875 | False | — | 76/0 | 0.0017 |
| memory_positive | 7 | valid | TP | 0.6631 | 0.1164 | 1.0611 | 1.1775 | True | 0.0025041480000140837 | 76/0 | 0.0011 |
| memory_positive | 4 | valid | TP | 0.6631 | 0.1924 | 1.061 | 1.2534 | True | 0.0017104560000120728 | 76/0 | 0.0029 |
| memory_positive | 8 | valid | TP | 0.663 | 0.2727 | 1.0609 | 1.3336 | True | 29.999873870999977 | 76/0 | 0.0011 |
| memory_positive | 10 | valid | TP | 0.663 | 0.2664 | 1.061 | 1.3274 | True | 29.998319451999976 | 76/0 | 0.0056 |
| cpu_positive | 5 | valid | TP | 0.9012 | 26.2672 | 0.0052 | 26.2724 | True | 30.00021700000002 | 76/0 | 0 |
| cpu_positive | 3 | valid | TP | 0.9021 | 26.2914 | 0.0052 | 26.2966 | True | 30.00219690199998 | 76/0 | 0.0009 |
| memory_positive | 1 | valid | TP | 0.663 | 0.2711 | 1.0609 | 1.332 | True | 0.0023010979999753545 | 76/0 | 0.0002 |
| cpu_positive | 9 | valid | TP | 0.9008 | 26.2547 | 0.0052 | 26.2599 | True | 0.0020154609999849527 | 76/0 | 0.0005 |
| cpu_positive | 1 | valid | TP | 0.9015 | 26.2759 | 0.0051 | 26.281 | True | 30.000758974999997 | 76/0 | 0.0005 |
| cpu_negative | 7 | valid | TN | 0.0174 | 0.4558 | 0.0051 | 0.4609 | False | — | 76/0 | 0.0006 |
| memory_positive | 2 | valid | TP | 0.6628 | 0.1783 | 1.0607 | 1.239 | True | 0.0012576560000070458 | 76/0 | 0.003 |
| cpu_negative | 1 | valid | TN | 0.0073 | 0.1901 | 0.0052 | 0.1953 | False | — | 76/0 | 0.0037 |
| memory_negative | 5 | valid | TN | 0 | 0.1887 | 0 | 0.1887 | False | — | 76/0 | 0.0006 |
| memory_negative | 7 | valid | TN | 0 | 0.2822 | 0 | 0.2822 | False | — | 76/0 | 0.008 |
| memory_negative | 6 | valid | TN | 0 | 0.1958 | 0 | 0.1958 | False | — | 76/0 | 0.0101 |
| memory_positive | 3 | valid | TP | 0.6631 | 0.2391 | 1.061 | 1.3001 | True | 29.993679815999997 | 76/0 | 0.0019 |
| cpu_positive | 6 | valid | TP | 0.9009 | 26.2576 | 0.0051 | 26.2627 | True | 0.002749375999997028 | 76/0 | 0.0003 |
| cpu_positive | 2 | valid | TP | 0.9017 | 26.2803 | 0.0052 | 26.2855 | True | 0.0023348390000137442 | 76/0 | 0.0005 |

## Matriz de confusão e taxas

| Cenário | Válidas | Inválidas | Erros | TP | FN | FP | TN | Taxa | Wilson 95% |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---|
| memory_positive | 10 | 0 | 0 | 10 | 0 | 0 | 0 | 1.0 | [0.7225, 1.0] |
| memory_negative | 10 | 0 | 0 | 0 | 0 | 0 | 10 | 0.0 | [0, 0.2775] |
| cpu_positive | 10 | 0 | 0 | 10 | 0 | 0 | 0 | 1.0 | [0.7225, 1.0] |
| cpu_negative | 10 | 0 | 0 | 0 | 0 | 0 | 10 | 0.0 | [0, 0.2775] |

## Incidentes e retries

- Primeira imagem incorporou `.env`; Dockerfile corrigido.
- Tag reaproveitada pelo runtime; adotadas nova tag e carga com overwrite.
- Chave do node-exporter mudou no chart; values corrigidos.
- Forwards em background morreram; usados forwards supervisionados.
- Primeira seleção do teste PostgreSQL não encontrou testes; repetida corretamente.
- Piloto incluiu namespaces de infraestrutura; filtros corrigidos antes da campanha final.
- Primeira rodada limpa revelou notificação do recurso secundário e amostra instantânea transitória; evidência preservada em `results/incidents`, manifests neutralizados e classificação alinhada à consolidação usada pelo tracker.
- O commit-base inicial avançou externamente para `333b330` durante a execução; a automação não executou commit e o metadado final registra o commit efetivamente observado.

## Artefatos e hashes SHA-256

Artefatos brutos: `tests/finops-eval/results`.

| Arquivo | SHA-256 |
|---|---|
| `results/cluster-objects.yaml` | `691b3d3493e775af96b7add37f2d681690c6cf55a1cf9f8f3226871df6096185` |
| `results/contracts.json` | `50020a60898fb768be0e1c1a2d43a39f4a2e8aa6e42ebbe9f181d0c2f34b0890` |
| `results/evidence/cpu_negative-01.json` | `8de354d42c7aa6285d19fec088b1e925a4c82c325085cbaf513755993315415e` |
| `results/evidence/cpu_negative-02.json` | `473541edfd24b6e0b1f67fcd0c05c3ee53c663a8a5d3cde9ac0f574a2a992cd4` |
| `results/evidence/cpu_negative-03.json` | `c17e73e0bba0063422da64fe13cd6159bd0bf55c16cd40537561479f6d169215` |
| `results/evidence/cpu_negative-04.json` | `94ea1f16b8784465a5f57cdf9fe27084573657e7265e66d084424c2efc2f15a8` |
| `results/evidence/cpu_negative-05.json` | `e757e890e5c3d604449e4d1c13730a278beada34fc0477e8f97b35d731d208a1` |
| `results/evidence/cpu_negative-06.json` | `159e7eaa4c8b48cfd8b9d081a15596b96683a136d5c3dd5f74142af276f641fb` |
| `results/evidence/cpu_negative-07.json` | `b9d4cecfd73e384e9939fecc9f1a84ad7ecb7d3843613ed701334f0c8a80a1f6` |
| `results/evidence/cpu_negative-08.json` | `dc15018d57c9ff08e4eb039b519e2e213fe4b8765038d2613f58609126421a26` |
| `results/evidence/cpu_negative-09.json` | `1e8bf996dba74e079150bf82fe56797f3b387799a64ada5412162df3639a8b4b` |
| `results/evidence/cpu_negative-10.json` | `509d8f754531442dd8df691b8ff72670444e23332531806356de02bd0b1aa5f5` |
| `results/evidence/cpu_positive-01.json` | `a6c8a4d45d288a7498f6e9573a711eb7b07187a4698d8f17c4ec829c46c1fe22` |
| `results/evidence/cpu_positive-02.json` | `9f830ed4929d44c2f5df270f40425cfcbca69e18cdfa2fda34a6f316d60d3898` |
| `results/evidence/cpu_positive-03.json` | `2e4badbee859c6f46da0f58ac299aa2a0e472bc1171d280224c415058d10abc2` |
| `results/evidence/cpu_positive-04.json` | `71b97d579dc631236ba2567ca5c36263bad2c787b2dd7401c484881757cdf4b3` |
| `results/evidence/cpu_positive-05.json` | `5376f7b92c3a55003ed03ee15703a977252b1396bddb4dc7cd3fb8809f3260ba` |
| `results/evidence/cpu_positive-06.json` | `b254710b6b1d65c29852c29a6eba1ee9fa3dc855e0cffa9869ba044fcdf2c634` |
| `results/evidence/cpu_positive-07.json` | `27be30aed0ea25480eebda2bad50b9c0ee98e22289dead530ee654540f346511` |
| `results/evidence/cpu_positive-08.json` | `7c63ccc91b5849633fa1235ba2ad7f2738e7a14fdf652df670d2ea5c73698b3d` |
| `results/evidence/cpu_positive-09.json` | `94f6768f30f48df933b6277b285bda123e079841c56b0799166ceaa598b0ab58` |
| `results/evidence/cpu_positive-10.json` | `cb5999df984a08f8babc278642f756bf292281f07e01eabdbfe67fc179b03784` |
| `results/evidence/memory_negative-01.json` | `f22dc28690d5568a60fadb016ce8ce1587e3f47853fd81265bfa2c857efa9356` |
| `results/evidence/memory_negative-02.json` | `bf2d1f91f48d0cca7419e0ca4dc80812c78a2eced29809877770aba9f4393494` |
| `results/evidence/memory_negative-03.json` | `5d0d5aed0242a14a082c87806dbe2dc242ad3a80617ba96deab1b6ed7f45dcb5` |
| `results/evidence/memory_negative-04.json` | `6f4d5b93e21da82d69bc54a7675f6a4cccc5a6ede5238272c64cef1e1627df67` |
| `results/evidence/memory_negative-05.json` | `e1cd6bb822d705d1b4e3c0d4b28e3631ce23ab739ee9268dce216a71538e4bbb` |
| `results/evidence/memory_negative-06.json` | `a2bf539e31f704013abaa95f372192666efa675b62a1433be194dcfe25ff8fe7` |
| `results/evidence/memory_negative-07.json` | `b7ad8007853e26f9f7a5b3ef377a98b92489f0c46ed1ae8358dbc266c55769c0` |
| `results/evidence/memory_negative-08.json` | `9587d1b1d5bcbf7165c31be8c9be720ad9911f477410e3153a1a8a6d8b9a6efc` |
| `results/evidence/memory_negative-09.json` | `314e6a0bf933e441ec9e92dfabc7ec2f7e3e4a142a6ac4250ef0fa9df83962f0` |
| `results/evidence/memory_negative-10.json` | `cc7f03acbc501170353d1c5e6a82aa77a41ee3ef9d3a1b7f23f2c253c4b60870` |
| `results/evidence/memory_positive-01.json` | `8f784954c686a5d9a6e80e0151171c765a6ece2ddb6e73a600cf6fa9101a3805` |
| `results/evidence/memory_positive-02.json` | `319dddc67f56b5ae08ef1c5e8f724b113b1b26f0f2ab3ca29a716d79a56f7d4f` |
| `results/evidence/memory_positive-03.json` | `74cd9d6663cb6daeec9c9e088e35c983a0d7e175c34d08ebbdd94625e892ee96` |
| `results/evidence/memory_positive-04.json` | `e1575bc993accacba432be619d744d9cedf77edecaf1c3f43e2fdc8f6fff0092` |
| `results/evidence/memory_positive-05.json` | `ba0aefc8d6f68b80b44423136a8619afd2997eab5f88a8fbcd80d3160dbe72a2` |
| `results/evidence/memory_positive-06.json` | `01e272d2bf6c2565e57e192abb9a7be0272c208d92d590ae4c885f2c0094bc37` |
| `results/evidence/memory_positive-07.json` | `03d29fc152175c70ea6b0e42bd5bae263fa7068fa1ce50a1e722975aa8853722` |
| `results/evidence/memory_positive-08.json` | `475fe27a445f44f0de3807b964d0dbe9272fb21eeb4cecc62b3254147a5666e5` |
| `results/evidence/memory_positive-09.json` | `78c71a232d9fe18f5e1b8264000769cda00b13e746b32f57c8416e4100c62e29` |
| `results/evidence/memory_positive-10.json` | `461f19dd35ba0891f9f6f3afb147757b8ad53129f1f72fd34d1f00adf0a24718` |
| `results/go-analyze-image.json` | `d4ef3e9c83ce54dcd2afa62686756c4ae161fc75e3d97c11bf0f0fcff0f3b005` |
| `results/helm-releases.json` | `b01cbb06da27a6a3483038b4ea451990c72f934ad1ac76a55880a6fa055876f7` |
| `results/incidents/pilot-cpu-negative-invalid.json` | `d74246368cddb0fcceeb80641bd512da5bc4a3cee161aed3e1d73bf630574b47` |
| `results/logs/campaign.log` | `9345525100f38fb5563d6fac6f228f544d7d3ce623955ef160beedca1652c21e` |
| `results/logs/contracts.log` | `8e05173a1c517c8f12e1cdf3e989cfd4451531444fbe44cbeb19369e85f5f9e7` |
| `results/logs/go-analyze.log` | `8d826d53cd3ed82014e2deb989cd59d30f0b91de15d30dc23475397db98172c2` |
| `results/logs/go-test-race.log` | `8951c929b3c95239e336fe8fabcd94c0d863c330acdcd7aff73910352d6c0520` |
| `results/logs/go-test.log` | `8951c929b3c95239e336fe8fabcd94c0d863c330acdcd7aff73910352d6c0520` |
| `results/logs/go-vet.log` | `01ba4719c80b6fe911b091a7c05124b64eeece964e09c058ef8f9805daca546b` |
| `results/logs/manifests-server-dry-run.log` | `b2fec7ac815e977629518b6a283d49916893cf8e556fff7030649a7abd7d2f07` |
| `results/logs/port-forward-go-analyze.log` | `aee64841c9b5f76e99d86399fadb749ab94a6f0f6f28a647aff54f764e864eae` |
| `results/logs/port-forward-postgres.log` | `e85bd3752947d8317e7e7e0491d86adc039612e8e81260f3a84a96cea135f68c` |
| `results/logs/port-forward-prometheus.log` | `b00df694c9c8209b1d4f05092da0015f4e177ea4b1d155514edbfea4ff31e955` |
| `results/logs/postgres-integration.log` | `103c915312a5c2b48eec97bf10bd53753acb6cf43dd7cbc2faff02a683b480a8` |
| `results/logs/python-tests.log` | `b124b89340b2cf23b8ee525e41b6dc23bf4a75ebe71c4a1f9933ffb15737dde2` |
| `results/logs/setup.log` | `4a42e7bab48755b50c760accbfaec65b33641398bd342a4a30e2c8b0d594165b` |
| `results/logs/validation-campaign-dry-run.log` | `31b56442641c460d61dcc5f73b70b010c23fd3485915a1f36bfc09c3f4668575` |
| `results/logs/validation-contracts.log` | `0bc6ebabc2e7ccc499f2a9b305c1e98bed047e63152c4819813f60c0571d0cd4` |
| `results/logs/validation-git-diff-check.log` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| `results/logs/validation-go-test-race.log` | `ba243df20914d08f66cd8361acce0136a814444633793265e21f27ab13080342` |
| `results/logs/validation-go-test.log` | `fd91bd28c85d3a447aae4ed3c2580f56d7aaa7044f226179a9b3b82fe19f18a1` |
| `results/logs/validation-go-vet.log` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| `results/logs/validation-manifests-server-dry-run.log` | `b2fec7ac815e977629518b6a283d49916893cf8e556fff7030649a7abd7d2f07` |
| `results/logs/validation-postgres-integration.log` | `9c4a26741c6816506577f409e11566212d87211c17e68138ea7585ff3a857f3d` |
| `results/logs/validation-python-compile.log` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| `results/logs/validation-python-tests.log` | `968f9cbb7d897f12536b9f01664cf3782c370612dd7c9f3197ad1fe114e694b1` |
| `results/metadata.json` | `31f043ed19255ef2becbdf10b15bc4de7a08e3333e4541ee07a7e1bfdb9bddc7` |
| `results/per_run.csv` | `7c1afecb21f55a47775ce1bfb749e82699f27456e835029478f791308770df1b` |
| `results/pods.txt` | `7e7097c274f03dc12419b0ee6e9426fcf194abf30e208847093631cddf4e347d` |
| `results/preflight.json` | `c8a4aac4d5ba4d9c70f887c53e5b5809f6297067a6fb85047bba7b17b394646a` |
| `results/prometheus-targets.json` | `6068ceede08a93c8b6fb69eab43b51b8846aa31dd643c53017e4802ab917ab43` |
| `results/summary.csv` | `b581e815ac40bc23a8a2fd167dc859d9eec80e8a8d6e545a9e4acdf9a8255db8` |
| `results/validation.json` | `20e8402ea5aa3b16c32dc79fe32b2d6a0e39f4c341da7632ac03188a982ebefa` |

## Estado final

O perfil Minikube foi preservado e parado, não excluído. Nenhum segredo foi incluído neste relatório ou nos artefatos textuais.
