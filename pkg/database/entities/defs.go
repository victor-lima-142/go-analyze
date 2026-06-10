package entities

import (
	"context"
	"database/sql"
)

var DefaultDB *sql.DB

type Entity interface {
	Migrate(ctx context.Context) error
	Create(ctx context.Context) error
	Update(ctx context.Context) error
	Delete(ctx context.Context) error
	Read(ctx context.Context, filters map[string]any) error
	ReadAll(ctx context.Context, filters map[string]any) ([]Entity, error)
}
