package entities

import (
	"context"
)

type WorkloadItemSnapshotModel struct {
	id                             int
	scrapeID                       int
	namespace                      string
	pod                            string
	container                      string
	cpuRequestedCores              float64
	cpuUsedCores                   float64
	cpuLimitCores                  *float64
	memoryRequestedBytes           float64
	memoryUsedBytes                float64
	memoryLimitBytes               *float64
	cpuWasteRatio                  float64
	memWasteRatio                  float64
	cpuProjectedMonthlyWasteUSD    float64
	memoryProjectedMonthlyWasteUSD float64
	projectedMonthlyWasteUSD       float64
}

func NewWorkloadItemSnapshot(scrapeID int, ns, pod, container string, cpuReq, cpuUsed float64, cpuLimit *float64, memReq, memUsed float64, memLimit *float64, cpuWaste, memWaste, cpuProjected, memoryProjected, projected float64) *WorkloadItemSnapshotModel {
	return &WorkloadItemSnapshotModel{
		scrapeID:                       scrapeID,
		namespace:                      ns,
		pod:                            pod,
		container:                      container,
		cpuRequestedCores:              cpuReq,
		cpuUsedCores:                   cpuUsed,
		cpuLimitCores:                  cpuLimit,
		memoryRequestedBytes:           memReq,
		memoryUsedBytes:                memUsed,
		memoryLimitBytes:               memLimit,
		cpuWasteRatio:                  cpuWaste,
		memWasteRatio:                  memWaste,
		cpuProjectedMonthlyWasteUSD:    cpuProjected,
		memoryProjectedMonthlyWasteUSD: memoryProjected,
		projectedMonthlyWasteUSD:       projected,
	}
}

// Getters and Setters
func (w *WorkloadItemSnapshotModel) ID() int      { return w.id }
func (w *WorkloadItemSnapshotModel) SetID(id int) { w.id = id }

func (w *WorkloadItemSnapshotModel) ScrapeID() int      { return w.scrapeID }
func (w *WorkloadItemSnapshotModel) SetScrapeID(id int) { w.scrapeID = id }

func (w *WorkloadItemSnapshotModel) Namespace() string       { return w.namespace }
func (w *WorkloadItemSnapshotModel) SetNamespace(val string) { w.namespace = val }

func (w *WorkloadItemSnapshotModel) Pod() string       { return w.pod }
func (w *WorkloadItemSnapshotModel) SetPod(val string) { w.pod = val }

func (w *WorkloadItemSnapshotModel) Container() string       { return w.container }
func (w *WorkloadItemSnapshotModel) SetContainer(val string) { w.container = val }

func (w *WorkloadItemSnapshotModel) CPURequestedCores() float64       { return w.cpuRequestedCores }
func (w *WorkloadItemSnapshotModel) SetCPURequestedCores(val float64) { w.cpuRequestedCores = val }

func (w *WorkloadItemSnapshotModel) CPUUsedCores() float64       { return w.cpuUsedCores }
func (w *WorkloadItemSnapshotModel) SetCPUUsedCores(val float64) { w.cpuUsedCores = val }

func (w *WorkloadItemSnapshotModel) CPULimitCores() *float64       { return w.cpuLimitCores }
func (w *WorkloadItemSnapshotModel) SetCPULimitCores(val *float64) { w.cpuLimitCores = val }

func (w *WorkloadItemSnapshotModel) MemoryRequestedBytes() float64 { return w.memoryRequestedBytes }
func (w *WorkloadItemSnapshotModel) SetMemoryRequestedBytes(val float64) {
	w.memoryRequestedBytes = val
}

func (w *WorkloadItemSnapshotModel) MemoryUsedBytes() float64       { return w.memoryUsedBytes }
func (w *WorkloadItemSnapshotModel) SetMemoryUsedBytes(val float64) { w.memoryUsedBytes = val }

func (w *WorkloadItemSnapshotModel) MemoryLimitBytes() *float64       { return w.memoryLimitBytes }
func (w *WorkloadItemSnapshotModel) SetMemoryLimitBytes(val *float64) { w.memoryLimitBytes = val }

func (w *WorkloadItemSnapshotModel) CPUWasteRatio() float64       { return w.cpuWasteRatio }
func (w *WorkloadItemSnapshotModel) SetCPUWasteRatio(val float64) { w.cpuWasteRatio = val }

func (w *WorkloadItemSnapshotModel) MemWasteRatio() float64       { return w.memWasteRatio }
func (w *WorkloadItemSnapshotModel) SetMemWasteRatio(val float64) { w.memWasteRatio = val }
func (w *WorkloadItemSnapshotModel) CPUProjectedMonthlyWasteUSD() float64 {
	return w.cpuProjectedMonthlyWasteUSD
}
func (w *WorkloadItemSnapshotModel) SetCPUProjectedMonthlyWasteUSD(v float64) {
	w.cpuProjectedMonthlyWasteUSD = v
}
func (w *WorkloadItemSnapshotModel) MemoryProjectedMonthlyWasteUSD() float64 {
	return w.memoryProjectedMonthlyWasteUSD
}
func (w *WorkloadItemSnapshotModel) SetMemoryProjectedMonthlyWasteUSD(v float64) {
	w.memoryProjectedMonthlyWasteUSD = v
}

func (w *WorkloadItemSnapshotModel) ProjectedMonthlyWasteUSD() float64 {
	return w.projectedMonthlyWasteUSD
}
func (w *WorkloadItemSnapshotModel) SetProjectedMonthlyWasteUSD(val float64) {
	w.projectedMonthlyWasteUSD = val
}

// Entity Interface Implementation
func (w *WorkloadItemSnapshotModel) Migrate(ctx context.Context) error {
	q := `CREATE TABLE IF NOT EXISTS workload_item_snapshots (
		id SERIAL PRIMARY KEY,
		scrape_id INT REFERENCES scrapes(id) ON DELETE CASCADE,
		namespace VARCHAR(253) NOT NULL,
		pod VARCHAR(253) NOT NULL,
		container VARCHAR(253) NOT NULL,
		cpu_requested_cores NUMERIC(12,4) NOT NULL,
		cpu_used_cores NUMERIC(12,4) NOT NULL,
		cpu_limit_cores NUMERIC(12,4),
		memory_requested_bytes NUMERIC(20,4) NOT NULL,
		memory_used_bytes NUMERIC(20,4) NOT NULL,
		memory_limit_bytes NUMERIC(20,4),
		cpu_waste_ratio NUMERIC(5,4) NOT NULL,
		mem_waste_ratio NUMERIC(5,4) NOT NULL,
		cpu_projected_monthly_waste_usd NUMERIC(12,4) NOT NULL,
		memory_projected_monthly_waste_usd NUMERIC(12,4) NOT NULL,
		projected_monthly_waste_usd NUMERIC(12,4) NOT NULL
	);`
	_, err := DefaultDB.ExecContext(ctx, q)
	if err != nil {
		return err
	}
	idxQ := `CREATE INDEX IF NOT EXISTS idx_workload_lookup ON workload_item_snapshots(namespace, pod, container);`
	_, err = DefaultDB.ExecContext(ctx, idxQ)
	return err
}

func (w *WorkloadItemSnapshotModel) Create(ctx context.Context) error {
	q := `INSERT INTO workload_item_snapshots (
		scrape_id, namespace, pod, container, cpu_requested_cores, cpu_used_cores, cpu_limit_cores,
		memory_requested_bytes, memory_used_bytes, memory_limit_bytes, cpu_waste_ratio, mem_waste_ratio,
		cpu_projected_monthly_waste_usd, memory_projected_monthly_waste_usd, projected_monthly_waste_usd
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) RETURNING id`
	return DefaultDB.QueryRowContext(ctx, q,
		w.scrapeID, w.namespace, w.pod, w.container, w.cpuRequestedCores, w.cpuUsedCores, w.cpuLimitCores,
		w.memoryRequestedBytes, w.memoryUsedBytes, w.memoryLimitBytes, w.cpuWasteRatio, w.memWasteRatio,
		w.cpuProjectedMonthlyWasteUSD, w.memoryProjectedMonthlyWasteUSD, w.projectedMonthlyWasteUSD).Scan(&w.id)
}

func (w *WorkloadItemSnapshotModel) Update(ctx context.Context) error {
	q := `UPDATE workload_item_snapshots SET
		scrape_id = $1, namespace = $2, pod = $3, container = $4, cpu_requested_cores = $5, cpu_used_cores = $6,
		cpu_limit_cores = $7, memory_requested_bytes = $8, memory_used_bytes = $9, memory_limit_bytes = $10,
		cpu_waste_ratio = $11, mem_waste_ratio = $12, cpu_projected_monthly_waste_usd = $13,
		memory_projected_monthly_waste_usd = $14, projected_monthly_waste_usd = $15 WHERE id = $16`
	_, err := DefaultDB.ExecContext(ctx, q,
		w.scrapeID, w.namespace, w.pod, w.container, w.cpuRequestedCores, w.cpuUsedCores, w.cpuLimitCores,
		w.memoryRequestedBytes, w.memoryUsedBytes, w.memoryLimitBytes, w.cpuWasteRatio, w.memWasteRatio,
		w.cpuProjectedMonthlyWasteUSD, w.memoryProjectedMonthlyWasteUSD, w.projectedMonthlyWasteUSD, w.id)
	return err
}

func (w *WorkloadItemSnapshotModel) Delete(ctx context.Context) error {
	q := `DELETE FROM workload_item_snapshots WHERE id = $1`
	_, err := DefaultDB.ExecContext(ctx, q, w.id)
	return err
}

func (w *WorkloadItemSnapshotModel) Read(ctx context.Context, filters map[string]any) error {
	where, args := buildWhereClause(filters)
	q := `SELECT id, scrape_id, namespace, pod, container, cpu_requested_cores, cpu_used_cores, cpu_limit_cores,
		memory_requested_bytes, memory_used_bytes, memory_limit_bytes, cpu_waste_ratio, mem_waste_ratio,
		cpu_projected_monthly_waste_usd, memory_projected_monthly_waste_usd, projected_monthly_waste_usd FROM workload_item_snapshots` + where + ` LIMIT 1`
	return DefaultDB.QueryRowContext(ctx, q, args...).Scan(
		&w.id, &w.scrapeID, &w.namespace, &w.pod, &w.container, &w.cpuRequestedCores, &w.cpuUsedCores, &w.cpuLimitCores,
		&w.memoryRequestedBytes, &w.memoryUsedBytes, &w.memoryLimitBytes, &w.cpuWasteRatio, &w.memWasteRatio,
		&w.cpuProjectedMonthlyWasteUSD, &w.memoryProjectedMonthlyWasteUSD, &w.projectedMonthlyWasteUSD)
}

func (w *WorkloadItemSnapshotModel) ReadAll(ctx context.Context, filters map[string]any) ([]Entity, error) {
	where, args := buildWhereClause(filters)
	q := `SELECT id, scrape_id, namespace, pod, container, cpu_requested_cores, cpu_used_cores, cpu_limit_cores,
		memory_requested_bytes, memory_used_bytes, memory_limit_bytes, cpu_waste_ratio, mem_waste_ratio,
		cpu_projected_monthly_waste_usd, memory_projected_monthly_waste_usd, projected_monthly_waste_usd FROM workload_item_snapshots` + where + ` ORDER BY namespace, pod, container`
	rows, err := DefaultDB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Entity
	for rows.Next() {
		item := &WorkloadItemSnapshotModel{}
		err := rows.Scan(
			&item.id, &item.scrapeID, &item.namespace, &item.pod, &item.container, &item.cpuRequestedCores, &item.cpuUsedCores, &item.cpuLimitCores,
			&item.memoryRequestedBytes, &item.memoryUsedBytes, &item.memoryLimitBytes, &item.cpuWasteRatio, &item.memWasteRatio,
			&item.cpuProjectedMonthlyWasteUSD, &item.memoryProjectedMonthlyWasteUSD, &item.projectedMonthlyWasteUSD)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

// Static helper mapping
type workloadItemSnapshotStatic struct{}

var WorkloadItemSnapshot workloadItemSnapshotStatic

func (workloadItemSnapshotStatic) Migrate(ctx context.Context) error {
	m := &WorkloadItemSnapshotModel{}
	return m.Migrate(ctx)
}

func (workloadItemSnapshotStatic) Create(ctx context.Context, w *WorkloadItemSnapshotModel) error {
	return w.Create(ctx)
}

func (workloadItemSnapshotStatic) Update(ctx context.Context, w *WorkloadItemSnapshotModel) error {
	return w.Update(ctx)
}

func (workloadItemSnapshotStatic) Delete(ctx context.Context, w *WorkloadItemSnapshotModel) error {
	return w.Delete(ctx)
}

func (workloadItemSnapshotStatic) Read(ctx context.Context, w *WorkloadItemSnapshotModel, filters map[string]any) error {
	return w.Read(ctx, filters)
}

func (workloadItemSnapshotStatic) ReadAll(ctx context.Context, filters map[string]any) ([]*WorkloadItemSnapshotModel, error) {
	m := &WorkloadItemSnapshotModel{}
	entities, err := m.ReadAll(ctx, filters)
	if err != nil {
		return nil, err
	}
	res := make([]*WorkloadItemSnapshotModel, len(entities))
	for i, e := range entities {
		res[i] = e.(*WorkloadItemSnapshotModel)
	}
	return res, nil
}
