BINARY_NAME=go-analyze
BINARY_SCRAPER=go-analyze-scraper
BINARY_SERVER=go-analyze-server
DOCKER_IMAGE=go-analyze
DOCKER_CONTAINER=go-analyze-container
PORT=8080
DB_CONTAINER=postgres-go-analyze

.DEFAULT_GOAL := help

.PHONY: help setup run-local run-scraper run-server build-local build-scraper build-server test lint clean docker-build docker-run docker-run-it docker-stop docker-logs postgres-run postgres-stop postgres-shell postgres-logs postgres-clean clear-ports clear-run clear-run-local swagger

help: ## Exibe este menu de ajuda interativa com todos os comandos disponíveis
	@echo "\033[1;35m=========================================================================\033[0m"
	@echo "🤖 \033[1;33mHELPERS DO MAKEFILE - RESOURCE ANALYZER\033[0m"
	@echo "\033[1;35m=========================================================================\033[0m"
	@echo "Comandos disponíveis:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[1;36m%-18s\033[0m %s\n", $$1, $$2}'
	@echo "\033[1;35m=========================================================================\033[0m"

setup: ## Configura o ambiente inicial criando o arquivo .env a partir do template
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "\033[1;32m✓ Arquivo .env criado com sucesso a partir de .env.example!\033[0m"; \
		echo "\033[1;33m👉 Edite o arquivo .env para adicionar suas credenciais reais do Prometheus.\033[0m"; \
	else \
		echo "\033[1;34mℹ Arquivo .env já existe no diretório root.\033[0m"; \
	fi

run-local: postgres-run ## Executa scraper + server juntos (comportamento padrão)
	@if [ ! -f .env ]; then \
		echo "\033[1;31m❌ Erro: Arquivo .env não encontrado!\033[0m"; \
		echo "👉 Execute '\033[1;36mmake setup\033[0m' primeiro para criar um template."; \
		exit 1; \
	fi
	@echo "\033[1;32m⚡ Carregando variáveis do .env e iniciando aplicação localmente...\033[0m"
	@set -a; [ -f .env ] && . ./.env; set +a; go run cmd/main.go

run-scraper: postgres-run ## Executa apenas o scraper (coleta métricas e persiste no banco, sem HTTP)
	@if [ ! -f .env ]; then \
		echo "\033[1;31m❌ Erro: Arquivo .env não encontrado!\033[0m"; \
		exit 1; \
	fi
	@echo "\033[1;32m⚡ Iniciando scraper...\033[0m"
	@set -a; [ -f .env ] && . ./.env; set +a; go run cmd/scraper/main.go

run-server: postgres-run ## Executa apenas o servidor HTTP (lê do banco, sem scraping)
	@if [ ! -f .env ]; then \
		echo "\033[1;31m❌ Erro: Arquivo .env não encontrado!\033[0m"; \
		exit 1; \
	fi
	@echo "\033[1;32m⚡ Iniciando servidor HTTP...\033[0m"
	@set -a; [ -f .env ] && . ./.env; set +a; go run cmd/server/main.go

build-local: ## Compila o binário go-analyze (scraper + server juntos)
	@echo "\033[1;32m🔨 Compilando o binário local...\033[0m"
	@mkdir -p bin
	@go build -o bin/$(BINARY_NAME) ./cmd/main.go
	@echo "\033[1;32m✓ Binário compilado com sucesso em bin/$(BINARY_NAME)!\033[0m"

build-scraper: ## Compila apenas o binário do scraper
	@echo "\033[1;32m🔨 Compilando o scraper...\033[0m"
	@mkdir -p bin
	@go build -o bin/$(BINARY_SCRAPER) ./cmd/scraper/main.go
	@echo "\033[1;32m✓ Binário compilado com sucesso em bin/$(BINARY_SCRAPER)!\033[0m"

build-server: ## Compila apenas o binário do servidor HTTP
	@echo "\033[1;32m🔨 Compilando o servidor...\033[0m"
	@mkdir -p bin
	@go build -o bin/$(BINARY_SERVER) ./cmd/server/main.go
	@echo "\033[1;32m✓ Binário compilado com sucesso em bin/$(BINARY_SERVER)!\033[0m"

test: ## Executa a suíte de testes unitários da aplicação
	@echo "\033[1;32m🧪 Executando os testes unitários...\033[0m"
	@go test -v ./...

lint: ## Executa o golangci-lint conforme .golangci.yml
	@echo "\033[1;32m🔍 Rodando golangci-lint...\033[0m"
	@golangci-lint run ./...

clean: ## Limpa binários compilados e arquivos temporários da aplicação
	@echo "\033[1;33m🧹 Limpando binários e arquivos temporários...\033[0m"
	@rm -rf bin/
	@go clean
	@echo "\033[1;32m✓ Limpeza concluída!\033[0m"

docker-build: ## Compila a imagem Docker da aplicação go-analyze
	@echo "\033[1;32m🐳 Construindo imagem Docker '$(DOCKER_IMAGE):latest'...\033[0m"
	@docker build -t $(DOCKER_IMAGE):latest .
	@echo "\033[1;32m✓ Imagem construída com sucesso!\033[0m"

docker-run: docker-build docker-stop ## Inicializa a aplicação em segundo plano dentro de um container Docker
	@echo "\033[1;32m🚀 Iniciando container Docker em segundo plano...\033[0m"
	@docker run -d \
		--name $(DOCKER_CONTAINER) \
		-p $(PORT):8080 \
		$(DOCKER_IMAGE):latest
	@echo "\033[1;32m✓ Container rodando em segundo plano na porta http://localhost:$(PORT)!\033[0m"
	@echo "👉 Use '\033[1;36mmake docker-logs\033[0m' para acompanhar os logs em tempo real."

docker-run-it: docker-build docker-stop ## Inicializa a aplicação em modo interativo dentro de um container Docker
	@echo "\033[1;32m🚀 Iniciando container Docker em modo interativo...\033[0m"
	@docker run --rm -it \
		--name $(DOCKER_CONTAINER) \
		-p $(PORT):8080 \
		$(DOCKER_IMAGE):latest

docker-stop: ## Para e remove o container Docker da aplicação go-analyze
	@if [ $$(docker ps -aq -f name=$(DOCKER_CONTAINER)) ]; then \
		echo "\033[1;33m🛑 Parando e removendo container antigo '$(DOCKER_CONTAINER)'...\033[0m"; \
		docker stop $(DOCKER_CONTAINER) > /dev/null 2>&1 || true; \
		docker rm $(DOCKER_CONTAINER) > /dev/null 2>&1 || true; \
		echo "\033[1;32m✓ Container antigo removido!\033[0m"; \
	fi

docker-logs: ## Exibe os logs em tempo real do container da aplicação go-analyze
	@echo "\033[1;32m📋 Exibindo logs em tempo real para '$(DOCKER_CONTAINER)'...\033[0m"
	@docker logs -f $(DOCKER_CONTAINER)

postgres-run: ## Inicializa o PostgreSQL no Docker com as credenciais padrão do .env e aguarda até estar pronto
	@if [ $$(docker ps -q -f name=$(DB_CONTAINER)) ]; then \
		echo "\033[1;34mℹ O container '$(DB_CONTAINER)' já está rodando.\033[0m"; \
	elif [ $$(docker ps -aq -f name=$(DB_CONTAINER)) ]; then \
		echo "\033[1;33m🛑 Container '$(DB_CONTAINER)' existe mas está parado. Iniciando...\033[0m"; \
		docker start $(DB_CONTAINER) > /dev/null; \
		echo "\033[1;32m✓ Container '$(DB_CONTAINER)' iniciado com sucesso!\033[0m"; \
	else \
		echo "\033[1;32m🐳 Criando e iniciando o container do PostgreSQL '$(DB_CONTAINER)'...\033[0m"; \
		docker run --name $(DB_CONTAINER) \
			-e POSTGRES_USER=postgres \
			-e POSTGRES_PASSWORD=postgres \
			-e POSTGRES_DB=go_analyze \
			-p 5432:5432 \
			-d postgres:15-alpine > /dev/null; \
		echo "\033[1;32m✓ Container PostgreSQL '$(DB_CONTAINER)' criado e rodando na porta 5432!\033[0m"; \
	fi
	@echo "\033[1;33m⏳ Aguardando o PostgreSQL estar pronto para receber conexões...\033[0m"; \
	for i in {1..15}; do \
		if docker exec $(DB_CONTAINER) pg_isready -U postgres -d go_analyze >/dev/null 2>&1; then \
			echo "\033[1;32m✓ PostgreSQL está pronto e aceitando conexões!\033[0m"; \
			exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "\033[1;31m⚠️  Aviso: O PostgreSQL demorou muito para responder. Pode ser que ainda não esteja totalmente pronto.\033[0m"; \

postgres-stop: ## Para o container Docker do PostgreSQL (mantendo os dados do container)
	@if [ $$(docker ps -q -f name=$(DB_CONTAINER)) ]; then \
		echo "\033[1;33m🛑 Parando o container '$(DB_CONTAINER)'...\033[0m"; \
		docker stop $(DB_CONTAINER) > /dev/null; \
		echo "\033[1;32m✓ Container parado!\033[0m"; \
	else \
		echo "\033[1;34mℹ O container '$(DB_CONTAINER)' não está rodando.\033[0m"; \
	fi

postgres-shell: ## Abre o terminal psql interativo dentro do container do PostgreSQL
	@if [ ! $$(docker ps -q -f name=$(DB_CONTAINER)) ]; then \
		echo "\033[1;31m❌ Erro: O container '$(DB_CONTAINER)' não está rodando!\033[0m"; \
		echo "👉 Execute '\033[1;36mmake postgres-run\033[0m' primeiro."; \
		exit 1; \
	fi
	@docker exec -it $(DB_CONTAINER) psql -U postgres -d go_analyze

postgres-logs: ## Exibe os logs em tempo real do container do PostgreSQL
	@echo "\033[1;32m📋 Exibindo logs em tempo real para '$(DB_CONTAINER)'...\033[0m"
	@docker logs -f $(DB_CONTAINER)

postgres-clean: ## Para e remove completamente o container do PostgreSQL do Docker
	@if [ $$(docker ps -aq -f name=$(DB_CONTAINER)) ]; then \
		echo "\033[1;33m🧹 Removendo o container '$(DB_CONTAINER)'...\033[0m"; \
		docker stop $(DB_CONTAINER) > /dev/null 2>&1 || true; \
		docker rm $(DB_CONTAINER) > /dev/null 2>&1 || true; \
		echo "\033[1;32m✓ Container do PostgreSQL removido com sucesso!\033[0m"; \
	else \
		echo "\033[1;34mℹ Nenhum container PostgreSQL com nome '$(DB_CONTAINER)' encontrado para remover.\033[0m"; \
	fi

clear-ports: ## Limpa port-forwards no kubectl e processos locais usando portas do Go (8080/8081)
	@echo "\033[1;33m🧹 Limpando processos locais do Go e port-forwards do kubectl...\033[0m"
	-@pkill -f "kubectl port-forward" 2>/dev/null; true
	-@pkill -f "go run cmd/main.go" 2>/dev/null; true
	-@pkill -f "go run cmd/scraper/main.go" 2>/dev/null; true
	-@pkill -f "go run cmd/server/main.go" 2>/dev/null; true
	-@pkill -f "bin/go-analyze" 2>/dev/null; true
	@lsof -t -i :8080 -i :8081 2>/dev/null | xargs -r kill -9 2>/dev/null; true
	@echo "\033[1;32m✓ Limpeza de portas locais e do kubectl finalizada!\033[0m"

clear-run: ## Para tudo, remove o Postgres, recria do zero e sobe a aplicação em Docker
	@$(MAKE) clear-ports
	@$(MAKE) docker-stop
	@$(MAKE) postgres-clean
	@$(MAKE) postgres-run
	@$(MAKE) docker-run
	@echo "\033[1;32m✓ Aplicação e banco recriados e iniciados em Docker do zero com sucesso!\033[0m"

clear-run-local: ## Para tudo, remove o Postgres, recria do zero e roda a aplicação localmente
	@$(MAKE) clear-ports
	@$(MAKE) postgres-clean
	@$(MAKE) postgres-run
	@$(MAKE) run-local

swagger: ## Gera a documentação OpenAPI/Swagger a partir dos comentários do código
	@echo "\033[1;32m📝 Gerando documentação OpenAPI/Swagger...\033[0m"
	@$(shell go env GOPATH)/bin/swag init -g cmd/main.go
	@echo "\033[1;32m✓ Documentação gerada com sucesso na pasta 'docs/'!\033[0m"

