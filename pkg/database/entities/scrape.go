package entities

import (
	"context"
	"fmt"
	"sort"
	"time"
)

type ScrapeModel struct {
	id         int
	scrapedAt  time.Time
	durationMs int
}

func NewScrape(durationMs int) *ScrapeModel {
	return &ScrapeModel{
		scrapedAt:  time.Now(),
		durationMs: durationMs,
	}
}

// Getters and Setters
func (s *ScrapeModel) ID() int      { return s.id }
func (s *ScrapeModel) SetID(id int) { s.id = id }

func (s *ScrapeModel) ScrapedAt() time.Time     { return s.scrapedAt }
func (s *ScrapeModel) SetScrapedAt(t time.Time) { s.scrapedAt = t }

func (s *ScrapeModel) DurationMs() int     { return s.durationMs }
func (s *ScrapeModel) SetDurationMs(d int) { s.durationMs = d }

// Entity Interface Implementation
func (s *ScrapeModel) Migrate(ctx context.Context) error {
	q := `CREATE TABLE IF NOT EXISTS scrapes (
		id SERIAL PRIMARY KEY,
		scraped_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
		duration_ms INT NOT NULL
	);`
	_, err := DefaultDB.ExecContext(ctx, q)
	return err
}

func (s *ScrapeModel) Create(ctx context.Context) error {
	q := `INSERT INTO scrapes (scraped_at, duration_ms) VALUES ($1, $2) RETURNING id`
	return DefaultDB.QueryRowContext(ctx, q, s.scrapedAt, s.durationMs).Scan(&s.id)
}

func (s *ScrapeModel) Update(ctx context.Context) error {
	q := `UPDATE scrapes SET scraped_at = $1, duration_ms = $2 WHERE id = $3`
	_, err := DefaultDB.ExecContext(ctx, q, s.scrapedAt, s.durationMs, s.id)
	return err
}

func (s *ScrapeModel) Delete(ctx context.Context) error {
	q := `DELETE FROM scrapes WHERE id = $1`
	_, err := DefaultDB.ExecContext(ctx, q, s.id)
	return err
}

func (s *ScrapeModel) Read(ctx context.Context, filters map[string]any) error {
	where, args := buildWhereClause(filters)
	q := `SELECT id, scraped_at, duration_ms FROM scrapes` + where + ` LIMIT 1`
	return DefaultDB.QueryRowContext(ctx, q, args...).Scan(&s.id, &s.scrapedAt, &s.durationMs)
}

func (s *ScrapeModel) ReadAll(ctx context.Context, filters map[string]any) ([]Entity, error) {
	where, args := buildWhereClause(filters)
	q := `SELECT id, scraped_at, duration_ms FROM scrapes` + where
	rows, err := DefaultDB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Entity
	for rows.Next() {
		item := &ScrapeModel{}
		if err := rows.Scan(&item.id, &item.scrapedAt, &item.durationMs); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

// Static helper mapping
type scrapeStatic struct{}

var Scrape scrapeStatic

func (scrapeStatic) Migrate(ctx context.Context) error {
	m := &ScrapeModel{}
	return m.Migrate(ctx)
}

func (scrapeStatic) Create(ctx context.Context, s *ScrapeModel) error {
	return s.Create(ctx)
}

func (scrapeStatic) Update(ctx context.Context, s *ScrapeModel) error {
	return s.Update(ctx)
}

func (scrapeStatic) Delete(ctx context.Context, s *ScrapeModel) error {
	return s.Delete(ctx)
}

func (scrapeStatic) Read(ctx context.Context, s *ScrapeModel, filters map[string]any) error {
	return s.Read(ctx, filters)
}

func (scrapeStatic) ReadAll(ctx context.Context, filters map[string]any) ([]*ScrapeModel, error) {
	s := &ScrapeModel{}
	entities, err := s.ReadAll(ctx, filters)
	if err != nil {
		return nil, err
	}
	res := make([]*ScrapeModel, len(entities))
	for i, e := range entities {
		res[i] = e.(*ScrapeModel)
	}
	return res, nil
}

func buildWhereClause(filters map[string]any) (string, []any) {
	if len(filters) == 0 {
		return "", nil
	}
	keys := make([]string, 0, len(filters))
	for k := range filters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	where := " WHERE "
	args := make([]any, 0, len(filters))
	for i, k := range keys {
		if i > 0 {
			where += " AND "
		}
		where += fmt.Sprintf("%s = $%d", k, i+1)
		args = append(args, filters[k])
	}
	return where, args
}
