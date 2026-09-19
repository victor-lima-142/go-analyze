package entities

import (
	"context"
)

type IndicatorSnapshotModel struct {
	id                             int
	scrapeID                       int
	cpuWasteRatio                  float64
	memWasteRatio                  float64
	cpuProjectedMonthlyWasteUSD    float64
	memoryProjectedMonthlyWasteUSD float64
	projectedMonthlyWasteUSD       float64
	totalCPURequested              float64
	totalCPUUsed                   float64
	totalMemoryRequested           float64
	totalMemoryUsed                float64
}

func NewIndicatorSnapshot(scrapeID int, cpuWaste, memWaste, cpuProjected, memoryProjected, projected, totalCPUReq, totalCPUUsed, totalMemReq, totalMemUsed float64) *IndicatorSnapshotModel {
	return &IndicatorSnapshotModel{
		scrapeID:                       scrapeID,
		cpuWasteRatio:                  cpuWaste,
		memWasteRatio:                  memWaste,
		cpuProjectedMonthlyWasteUSD:    cpuProjected,
		memoryProjectedMonthlyWasteUSD: memoryProjected,
		projectedMonthlyWasteUSD:       projected,
		totalCPURequested:              totalCPUReq,
		totalCPUUsed:                   totalCPUUsed,
		totalMemoryRequested:           totalMemReq,
		totalMemoryUsed:                totalMemUsed,
	}
}

// Getters and Setters
func (i *IndicatorSnapshotModel) ID() int      { return i.id }
func (i *IndicatorSnapshotModel) SetID(id int) { i.id = id }

func (i *IndicatorSnapshotModel) ScrapeID() int      { return i.scrapeID }
func (i *IndicatorSnapshotModel) SetScrapeID(id int) { i.scrapeID = id }

func (i *IndicatorSnapshotModel) CPUWasteRatio() float64       { return i.cpuWasteRatio }
func (i *IndicatorSnapshotModel) SetCPUWasteRatio(val float64) { i.cpuWasteRatio = val }

func (i *IndicatorSnapshotModel) MemWasteRatio() float64       { return i.memWasteRatio }
func (i *IndicatorSnapshotModel) SetMemWasteRatio(val float64) { i.memWasteRatio = val }

func (i *IndicatorSnapshotModel) CPUProjectedMonthlyWasteUSD() float64 {
	return i.cpuProjectedMonthlyWasteUSD
}
func (i *IndicatorSnapshotModel) SetCPUProjectedMonthlyWasteUSD(v float64) {
	i.cpuProjectedMonthlyWasteUSD = v
}
func (i *IndicatorSnapshotModel) MemoryProjectedMonthlyWasteUSD() float64 {
	return i.memoryProjectedMonthlyWasteUSD
}
func (i *IndicatorSnapshotModel) SetMemoryProjectedMonthlyWasteUSD(v float64) {
	i.memoryProjectedMonthlyWasteUSD = v
}

func (i *IndicatorSnapshotModel) ProjectedMonthlyWasteUSD() float64 {
	return i.projectedMonthlyWasteUSD
}
func (i *IndicatorSnapshotModel) SetProjectedMonthlyWasteUSD(val float64) {
	i.projectedMonthlyWasteUSD = val
}

func (i *IndicatorSnapshotModel) TotalCPURequested() float64       { return i.totalCPURequested }
func (i *IndicatorSnapshotModel) SetTotalCPURequested(val float64) { i.totalCPURequested = val }

func (i *IndicatorSnapshotModel) TotalCPUUsed() float64       { return i.totalCPUUsed }
func (i *IndicatorSnapshotModel) SetTotalCPUUsed(val float64) { i.totalCPUUsed = val }

func (i *IndicatorSnapshotModel) TotalMemoryRequested() float64       { return i.totalMemoryRequested }
func (i *IndicatorSnapshotModel) SetTotalMemoryRequested(val float64) { i.totalMemoryRequested = val }

func (i *IndicatorSnapshotModel) TotalMemoryUsed() float64       { return i.totalMemoryUsed }
func (i *IndicatorSnapshotModel) SetTotalMemoryUsed(val float64) { i.totalMemoryUsed = val }

// Entity Interface Implementation
func (i *IndicatorSnapshotModel) Migrate(ctx context.Context) error {
	q := `CREATE TABLE IF NOT EXISTS indicator_snapshots (
		id SERIAL PRIMARY KEY,
		scrape_id INT REFERENCES scrapes(id) ON DELETE CASCADE,
		cpu_waste_ratio NUMERIC(5,4) NOT NULL,
		mem_waste_ratio NUMERIC(5,4) NOT NULL,
		cpu_projected_monthly_waste_usd NUMERIC(12,4) NOT NULL,
		memory_projected_monthly_waste_usd NUMERIC(12,4) NOT NULL,
		projected_monthly_waste_usd NUMERIC(12,4) NOT NULL,
		total_cpu_requested NUMERIC(12,4) NOT NULL,
		total_cpu_used NUMERIC(12,4) NOT NULL,
		total_memory_requested_bytes NUMERIC(20,4) NOT NULL,
		total_memory_used_bytes NUMERIC(20,4) NOT NULL
	);`
	_, err := DefaultDB.ExecContext(ctx, q)
	return err
}

func (i *IndicatorSnapshotModel) Create(ctx context.Context) error {
	q := `INSERT INTO indicator_snapshots (
		scrape_id, cpu_waste_ratio, mem_waste_ratio, cpu_projected_monthly_waste_usd, memory_projected_monthly_waste_usd, projected_monthly_waste_usd,
		total_cpu_requested, total_cpu_used, total_memory_requested_bytes, total_memory_used_bytes
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`
	return DefaultDB.QueryRowContext(ctx, q,
		i.scrapeID, i.cpuWasteRatio, i.memWasteRatio, i.cpuProjectedMonthlyWasteUSD, i.memoryProjectedMonthlyWasteUSD, i.projectedMonthlyWasteUSD,
		i.totalCPURequested, i.totalCPUUsed, i.totalMemoryRequested, i.totalMemoryUsed).Scan(&i.id)
}

func (i *IndicatorSnapshotModel) Update(ctx context.Context) error {
	q := `UPDATE indicator_snapshots SET
		scrape_id = $1, cpu_waste_ratio = $2, mem_waste_ratio = $3, cpu_projected_monthly_waste_usd = $4, memory_projected_monthly_waste_usd = $5,
		projected_monthly_waste_usd = $6, total_cpu_requested = $7, total_cpu_used = $8,
		total_memory_requested_bytes = $9, total_memory_used_bytes = $10 WHERE id = $11`
	_, err := DefaultDB.ExecContext(ctx, q,
		i.scrapeID, i.cpuWasteRatio, i.memWasteRatio, i.cpuProjectedMonthlyWasteUSD, i.memoryProjectedMonthlyWasteUSD, i.projectedMonthlyWasteUSD,
		i.totalCPURequested, i.totalCPUUsed, i.totalMemoryRequested, i.totalMemoryUsed, i.id)
	return err
}

func (i *IndicatorSnapshotModel) Delete(ctx context.Context) error {
	q := `DELETE FROM indicator_snapshots WHERE id = $1`
	_, err := DefaultDB.ExecContext(ctx, q, i.id)
	return err
}

func (i *IndicatorSnapshotModel) Read(ctx context.Context, filters map[string]any) error {
	where, args := buildWhereClause(filters)
	q := `SELECT id, scrape_id, cpu_waste_ratio, mem_waste_ratio, cpu_projected_monthly_waste_usd, memory_projected_monthly_waste_usd, projected_monthly_waste_usd,
		total_cpu_requested, total_cpu_used, total_memory_requested_bytes, total_memory_used_bytes FROM indicator_snapshots` + where + ` LIMIT 1`
	return DefaultDB.QueryRowContext(ctx, q, args...).Scan(
		&i.id, &i.scrapeID, &i.cpuWasteRatio, &i.memWasteRatio, &i.cpuProjectedMonthlyWasteUSD, &i.memoryProjectedMonthlyWasteUSD, &i.projectedMonthlyWasteUSD,
		&i.totalCPURequested, &i.totalCPUUsed, &i.totalMemoryRequested, &i.totalMemoryUsed)
}

func (i *IndicatorSnapshotModel) ReadAll(ctx context.Context, filters map[string]any) ([]Entity, error) {
	where, args := buildWhereClause(filters)
	q := `SELECT id, scrape_id, cpu_waste_ratio, mem_waste_ratio, cpu_projected_monthly_waste_usd, memory_projected_monthly_waste_usd, projected_monthly_waste_usd,
		total_cpu_requested, total_cpu_used, total_memory_requested_bytes, total_memory_used_bytes FROM indicator_snapshots` + where
	rows, err := DefaultDB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Entity
	for rows.Next() {
		item := &IndicatorSnapshotModel{}
		err := rows.Scan(
			&item.id, &item.scrapeID, &item.cpuWasteRatio, &item.memWasteRatio, &item.cpuProjectedMonthlyWasteUSD, &item.memoryProjectedMonthlyWasteUSD, &item.projectedMonthlyWasteUSD,
			&item.totalCPURequested, &item.totalCPUUsed, &item.totalMemoryRequested, &item.totalMemoryUsed)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

// Static helper mapping
type indicatorSnapshotStatic struct{}

var IndicatorSnapshot indicatorSnapshotStatic

func (indicatorSnapshotStatic) Migrate(ctx context.Context) error {
	m := &IndicatorSnapshotModel{}
	return m.Migrate(ctx)
}

func (indicatorSnapshotStatic) Create(ctx context.Context, i *IndicatorSnapshotModel) error {
	return i.Create(ctx)
}

func (indicatorSnapshotStatic) Update(ctx context.Context, i *IndicatorSnapshotModel) error {
	return i.Update(ctx)
}

func (indicatorSnapshotStatic) Delete(ctx context.Context, i *IndicatorSnapshotModel) error {
	return i.Delete(ctx)
}

func (indicatorSnapshotStatic) Read(ctx context.Context, i *IndicatorSnapshotModel, filters map[string]any) error {
	return i.Read(ctx, filters)
}

func (indicatorSnapshotStatic) ReadAll(ctx context.Context, filters map[string]any) ([]*IndicatorSnapshotModel, error) {
	m := &IndicatorSnapshotModel{}
	entities, err := m.ReadAll(ctx, filters)
	if err != nil {
		return nil, err
	}
	res := make([]*IndicatorSnapshotModel, len(entities))
	for i, e := range entities {
		res[i] = e.(*IndicatorSnapshotModel)
	}
	return res, nil
}
