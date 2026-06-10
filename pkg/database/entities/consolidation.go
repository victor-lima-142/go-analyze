package entities

import (
	"context"
	"time"
)

type ConsolidationModel struct {
	id                       int
	consolidatedAt           time.Time
	startTime                time.Time
	endTime                  time.Time
	scrapesCount             int
	cpuWasteRatio            float64
	memWasteRatio            float64
	pvcWasteRatio            float64
	hpaEfficiency            float64
	projectedMonthlyWasteUSD float64
	totalCPURequested        float64
	totalCPUUsed             float64
	totalMemoryRequested     float64
	totalMemoryUsed          float64
	oomRiskScore             float64
}

func NewConsolidation(startTime, endTime time.Time, scrapesCount int, cpuWaste, memWaste, pvcWaste, hpaEff, projected float64, totalCPUReq, totalCPUUsed, totalMemReq, totalMemUsed float64) *ConsolidationModel {
	return &ConsolidationModel{
		consolidatedAt:           time.Now(),
		startTime:                startTime,
		endTime:                  endTime,
		scrapesCount:             scrapesCount,
		cpuWasteRatio:            cpuWaste,
		memWasteRatio:            memWaste,
		pvcWasteRatio:            pvcWaste,
		hpaEfficiency:            hpaEff,
		projectedMonthlyWasteUSD: projected,
		totalCPURequested:        totalCPUReq,
		totalCPUUsed:             totalCPUUsed,
		totalMemoryRequested:     totalMemReq,
		totalMemoryUsed:          totalMemUsed,
	}
}

// Getters and Setters
func (c *ConsolidationModel) ID() int      { return c.id }
func (c *ConsolidationModel) SetID(id int) { c.id = id }

func (c *ConsolidationModel) ConsolidatedAt() time.Time     { return c.consolidatedAt }
func (c *ConsolidationModel) SetConsolidatedAt(t time.Time) { c.consolidatedAt = t }

func (c *ConsolidationModel) StartTime() time.Time     { return c.startTime }
func (c *ConsolidationModel) SetStartTime(t time.Time) { c.startTime = t }

func (c *ConsolidationModel) EndTime() time.Time     { return c.endTime }
func (c *ConsolidationModel) SetEndTime(t time.Time) { c.endTime = t }

func (c *ConsolidationModel) ScrapesCount() int     { return c.scrapesCount }
func (c *ConsolidationModel) SetScrapesCount(n int) { c.scrapesCount = n }

func (c *ConsolidationModel) CPUWasteRatio() float64       { return c.cpuWasteRatio }
func (c *ConsolidationModel) SetCPUWasteRatio(val float64) { c.cpuWasteRatio = val }

func (c *ConsolidationModel) MemWasteRatio() float64       { return c.memWasteRatio }
func (c *ConsolidationModel) SetMemWasteRatio(val float64) { c.memWasteRatio = val }

func (c *ConsolidationModel) PVCWasteRatio() float64       { return c.pvcWasteRatio }
func (c *ConsolidationModel) SetPVCWasteRatio(val float64) { c.pvcWasteRatio = val }

func (c *ConsolidationModel) HPAEfficiency() float64       { return c.hpaEfficiency }
func (c *ConsolidationModel) SetHPAEfficiency(val float64) { c.hpaEfficiency = val }

func (c *ConsolidationModel) ProjectedMonthlyWasteUSD() float64 {
	return c.projectedMonthlyWasteUSD
}
func (c *ConsolidationModel) SetProjectedMonthlyWasteUSD(val float64) {
	c.projectedMonthlyWasteUSD = val
}

func (c *ConsolidationModel) TotalCPURequested() float64       { return c.totalCPURequested }
func (c *ConsolidationModel) SetTotalCPURequested(val float64) { c.totalCPURequested = val }

func (c *ConsolidationModel) TotalCPUUsed() float64       { return c.totalCPUUsed }
func (c *ConsolidationModel) SetTotalCPUUsed(val float64) { c.totalCPUUsed = val }

func (c *ConsolidationModel) TotalMemoryRequested() float64       { return c.totalMemoryRequested }
func (c *ConsolidationModel) SetTotalMemoryRequested(val float64) { c.totalMemoryRequested = val }

func (c *ConsolidationModel) TotalMemoryUsed() float64       { return c.totalMemoryUsed }
func (c *ConsolidationModel) SetTotalMemoryUsed(val float64) { c.totalMemoryUsed = val }

func (c *ConsolidationModel) OOMRiskScore() float64       { return c.oomRiskScore }
func (c *ConsolidationModel) SetOOMRiskScore(val float64) { c.oomRiskScore = val }

// Entity Interface Implementation
func (c *ConsolidationModel) Migrate(ctx context.Context) error {
	q := `CREATE TABLE IF NOT EXISTS consolidations (
		id SERIAL PRIMARY KEY,
		consolidated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
		start_time TIMESTAMP WITH TIME ZONE NOT NULL,
		end_time TIMESTAMP WITH TIME ZONE NOT NULL,
		scrapes_count INT NOT NULL,
		cpu_waste_ratio NUMERIC(5,4) NOT NULL,
		mem_waste_ratio NUMERIC(5,4) NOT NULL,
		pvc_waste_ratio NUMERIC(5,4) NOT NULL,
		hpa_efficiency NUMERIC(5,4) NOT NULL,
		projected_monthly_waste_usd NUMERIC(12,4) NOT NULL,
		total_cpu_requested NUMERIC(12,4) NOT NULL,
		total_cpu_used NUMERIC(12,4) NOT NULL,
		total_memory_requested_bytes NUMERIC(20,4) NOT NULL,
		total_memory_used_bytes NUMERIC(20,4) NOT NULL
	);`
	_, err := DefaultDB.ExecContext(ctx, q)
	return err
}

func (c *ConsolidationModel) Create(ctx context.Context) error {
	q := `INSERT INTO consolidations (
		consolidated_at, start_time, end_time, scrapes_count, cpu_waste_ratio, mem_waste_ratio,
		pvc_waste_ratio, hpa_efficiency, projected_monthly_waste_usd, total_cpu_requested, total_cpu_used,
		total_memory_requested_bytes, total_memory_used_bytes
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING id`
	return DefaultDB.QueryRowContext(ctx, q,
		c.consolidatedAt, c.startTime, c.endTime, c.scrapesCount, c.cpuWasteRatio, c.memWasteRatio,
		c.pvcWasteRatio, c.hpaEfficiency, c.projectedMonthlyWasteUSD, c.totalCPURequested, c.totalCPUUsed,
		c.totalMemoryRequested, c.totalMemoryUsed).Scan(&c.id)
}

func (c *ConsolidationModel) Update(ctx context.Context) error {
	q := `UPDATE consolidations SET
		consolidated_at = $1, start_time = $2, end_time = $3, scrapes_count = $4, cpu_waste_ratio = $5,
		mem_waste_ratio = $6, pvc_waste_ratio = $7, hpa_efficiency = $8, projected_monthly_waste_usd = $9,
		total_cpu_requested = $10, total_cpu_used = $11, total_memory_requested_bytes = $12, total_memory_used_bytes = $13
		WHERE id = $14`
	_, err := DefaultDB.ExecContext(ctx, q,
		c.consolidatedAt, c.startTime, c.endTime, c.scrapesCount, c.cpuWasteRatio, c.memWasteRatio,
		c.pvcWasteRatio, c.hpaEfficiency, c.projectedMonthlyWasteUSD, c.totalCPURequested, c.totalCPUUsed,
		c.totalMemoryRequested, c.totalMemoryUsed, c.id)
	return err
}

func (c *ConsolidationModel) Delete(ctx context.Context) error {
	q := `DELETE FROM consolidations WHERE id = $1`
	_, err := DefaultDB.ExecContext(ctx, q, c.id)
	return err
}

func (c *ConsolidationModel) Read(ctx context.Context, filters map[string]any) error {
	where, args := buildWhereClause(filters)
	q := `SELECT id, consolidated_at, start_time, end_time, scrapes_count, cpu_waste_ratio, mem_waste_ratio,
		pvc_waste_ratio, hpa_efficiency, projected_monthly_waste_usd, total_cpu_requested, total_cpu_used,
		total_memory_requested_bytes, total_memory_used_bytes FROM consolidations` + where + ` LIMIT 1`
	return DefaultDB.QueryRowContext(ctx, q, args...).Scan(
		&c.id, &c.consolidatedAt, &c.startTime, &c.endTime, &c.scrapesCount, &c.cpuWasteRatio, &c.memWasteRatio,
		&c.pvcWasteRatio, &c.hpaEfficiency, &c.projectedMonthlyWasteUSD, &c.totalCPURequested, &c.totalCPUUsed,
		&c.totalMemoryRequested, &c.totalMemoryUsed)
}

func (c *ConsolidationModel) ReadAll(ctx context.Context, filters map[string]any) ([]Entity, error) {
	where, args := buildWhereClause(filters)
	q := `SELECT id, consolidated_at, start_time, end_time, scrapes_count, cpu_waste_ratio, mem_waste_ratio,
		pvc_waste_ratio, hpa_efficiency, projected_monthly_waste_usd, total_cpu_requested, total_cpu_used,
		total_memory_requested_bytes, total_memory_used_bytes FROM consolidations` + where + ` ORDER BY consolidated_at DESC`
	rows, err := DefaultDB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Entity
	for rows.Next() {
		item := &ConsolidationModel{}
		err := rows.Scan(
			&item.id, &item.consolidatedAt, &item.startTime, &item.endTime, &item.scrapesCount, &item.cpuWasteRatio, &item.memWasteRatio,
			&item.pvcWasteRatio, &item.hpaEfficiency, &item.projectedMonthlyWasteUSD, &item.totalCPURequested, &item.totalCPUUsed,
			&item.totalMemoryRequested, &item.totalMemoryUsed)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

// Static helper mapping
type consolidationStatic struct{}

var Consolidation consolidationStatic

func (consolidationStatic) Migrate(ctx context.Context) error {
	m := &ConsolidationModel{}
	return m.Migrate(ctx)
}

func (consolidationStatic) Create(ctx context.Context, c *ConsolidationModel) error {
	return c.Create(ctx)
}

func (consolidationStatic) Update(ctx context.Context, c *ConsolidationModel) error {
	return c.Update(ctx)
}

func (consolidationStatic) Delete(ctx context.Context, c *ConsolidationModel) error {
	return c.Delete(ctx)
}

func (consolidationStatic) Read(ctx context.Context, c *ConsolidationModel, filters map[string]any) error {
	return c.Read(ctx, filters)
}

func (consolidationStatic) ReadAll(ctx context.Context, filters map[string]any) ([]*ConsolidationModel, error) {
	m := &ConsolidationModel{}
	entities, err := m.ReadAll(ctx, filters)
	if err != nil {
		return nil, err
	}
	res := make([]*ConsolidationModel, len(entities))
	for i, e := range entities {
		res[i] = e.(*ConsolidationModel)
	}
	return res, nil
}
