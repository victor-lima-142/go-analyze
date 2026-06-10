# ADR 0008 — Padrão Active Record com Helpers Estáticos

## Status
Aceito (avaliado e mantido).

## Contexto
Cada entidade em `pkg/database/entities/` mistura: struct com campos privados, getters/setters, CRUD (`Migrate/Create/Update/Delete/Read/ReadAll`) e um helper estático (variável de pacote em maiúscula como `Scrape`, `IndicatorSnapshot`, etc.). O DB é uma variável global `DefaultDB *sql.DB`.

## Decisão
Manter o padrão atual em vez de migrar para um Repository pattern completo. Razões:
- O escopo do MBA não justifica um refactor invasivo de 5 entidades.
- A interface `DBService` foi expandida em `pkg/database/defs.go` para refletir todos os métodos cross-entity efetivamente usados, fechando o gap de testabilidade.
- A migração para Postgres é feita via `pkg/database/migrations/` (sistema versionado dedicado) — entidades não fazem mais `Migrate()` ad-hoc no startup.

## Alternativas consideradas
- **Repository pattern puro** (entidades imutáveis + DAOs): mais "go-idiomático", mas exigiria reescrever tudo.
- **GORM**: rejeitado — overhead desnecessário para 5 tabelas.

## Consequências
- Variável global `entities.DefaultDB` permanece, com setter dedicado pela camada `database.Connect`.
- Helpers estáticos não são mais usados para migração (delegado às migrations versionadas).
- Documentação para futuros mantenedores: o padrão é assumido, não acidental.
