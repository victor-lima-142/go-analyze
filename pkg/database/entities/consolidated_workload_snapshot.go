package entities

import (
	"context"
	"time"
)

type HistoricalWorkloadSnapshot struct {
	Snapshot       *ConsolidatedWorkloadSnapshotModel `json:"snapshot"`
	ConsolidatedAt time.Time                          `json:"consolidated_at"`
}

type ConsolidatedWorkloadSnapshotModel struct {
	id                       int
	consolidationID          int
	namespace                string
	pod                      string
	container                string
	cpuRequestedCores        float64
	cpuUsedCores             float64
	cpuLimitCores            *float64
	memoryRequestedBytes     float64
	memoryUsedBytes          float64
	memoryLimitBytes         *float64
	cpuWasteRatio            float64
	memWasteRatio            float64
	projectedMonthlyWasteUSD float64
	oomRiskScore             float64
}

func NewConsolidatedWorkloadSnapshot(consolidationID int, ns, pod, container string, cpuReq, cpuUsed float64, cpuLimit *float64, memReq, memUsed float64, memLimit *float64, cpuWaste, memWaste, projected float64) *ConsolidatedWorkloadSnapshotModel {
	return &ConsolidatedWorkloadSnapshotModel{
		consolidationID:          consolidationID,
		namespace:                ns,
		pod:                      pod,
		container:                container,
		cpuRequestedCores:        cpuReq,
		cpuUsedCores:             cpuUsed,
		cpuLimitCores:            cpuLimit,
		memoryRequestedBytes:     memReq,
		memoryUsedBytes:          memUsed,
		memoryLimitBytes:         memLimit,
		cpuWasteRatio:            cpuWaste,
		memWasteRatio:            memWaste,
		projectedMonthlyWasteUSD: projected,
	}
}

// Getters and Setters
func (w *ConsolidatedWorkloadSnapshotModel) ID() int      { return w.id }
func (w *ConsolidatedWorkloadSnapshotModel) SetID(id int) { w.id = id }

func (w *ConsolidatedWorkloadSnapshotModel) ConsolidationID() int      { return w.consolidationID }
func (w *ConsolidatedWorkloadSnapshotModel) SetConsolidationID(id int) { w.consolidationID = id }

func (w *ConsolidatedWorkloadSnapshotModel) Namespace() string       { return w.namespace }
func (w *ConsolidatedWorkloadSnapshotModel) SetNamespace(val string) { w.namespace = val }

func (w *ConsolidatedWorkloadSnapshotModel) Pod() string       { return w.pod }
func (w *ConsolidatedWorkloadSnapshotModel) SetPod(val string) { w.pod = val }

func (w *ConsolidatedWorkloadSnapshotModel) Container() string       { return w.container }
func (w *ConsolidatedWorkloadSnapshotModel) SetContainer(val string) { w.container = val }

func (w *ConsolidatedWorkloadSnapshotModel) CPURequestedCores() float64       { return w.cpuRequestedCores }
func (w *ConsolidatedWorkloadSnapshotModel) SetCPURequestedCores(val float64) { w.cpuRequestedCores = val }

func (w *ConsolidatedWorkloadSnapshotModel) CPUUsedCores() float64       { return w.cpuUsedCores }
func (w *ConsolidatedWorkloadSnapshotModel) SetCPUUsedCores(val float64) { w.cpuUsedCores = val }

func (w *ConsolidatedWorkloadSnapshotModel) CPULimitCores() *float64       { return w.cpuLimitCores }
func (w *ConsolidatedWorkloadSnapshotModel) SetCPULimitCores(val *float64) { w.cpuLimitCores = val }

func (w *ConsolidatedWorkloadSnapshotModel) MemoryRequestedBytes() float64 { return w.memoryRequestedBytes }
func (w *ConsolidatedWorkloadSnapshotModel) SetMemoryRequestedBytes(val float64) {
	w.memoryRequestedBytes = val
}

func (w *ConsolidatedWorkloadSnapshotModel) MemoryUsedBytes() float64       { return w.memoryUsedBytes }
func (w *ConsolidatedWorkloadSnapshotModel) SetMemoryUsedBytes(val float64) { w.memoryUsedBytes = val }

func (w *ConsolidatedWorkloadSnapshotModel) MemoryLimitBytes() *float64       { return w.memoryLimitBytes }
func (w *ConsolidatedWorkloadSnapshotModel) SetMemoryLimitBytes(val *float64) { w.memoryLimitBytes = val }

func (w *ConsolidatedWorkloadSnapshotModel) CPUWasteRatio() float64       { return w.cpuWasteRatio }
func (w *ConsolidatedWorkloadSnapshotModel) SetCPUWasteRatio(val float64) { w.cpuWasteRatio = val }

func (w *ConsolidatedWorkloadSnapshotModel) MemWasteRatio() float64       { return w.memWasteRatio }
func (w *ConsolidatedWorkloadSnapshotModel) SetMemWasteRatio(val float64) { w.memWasteRatio = val }

func (w *ConsolidatedWorkloadSnapshotModel) ProjectedMonthlyWasteUSD() float64 {
	return w.projectedMonthlyWasteUSD
}
func (w *ConsolidatedWorkloadSnapshotModel) SetProjectedMonthlyWasteUSD(val float64) {
	w.projectedMonthlyWasteUSD = val
}

func (w *ConsolidatedWorkloadSnapshotModel) OOMRiskScore() float64       { return w.oomRiskScore }
func (w *ConsolidatedWorkloadSnapshotModel) SetOOMRiskScore(val float64) { w.oomRiskScore = val }

// Entity Interface Implementation
func (w *ConsolidatedWorkloadSnapshotModel) Migrate(ctx context.Context) error {
	q := `CREATE TABLE IF NOT EXISTS consolidated_workload_snapshots (
		id SERIAL PRIMARY KEY,
		consolidation_id INT REFERENCES consolidations(id) ON DELETE CASCADE,
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
		projected_monthly_waste_usd NUMERIC(12,4) NOT NULL
	);`
	_, err := DefaultDB.ExecContext(ctx, q)
	if err != nil {
		return err
	}
	idxQ := `CREATE INDEX IF NOT EXISTS idx_consolidated_workload_lookup ON consolidated_workload_snapshots(namespace, pod, container);`
	_, err = DefaultDB.ExecContext(ctx, idxQ)
	return err
}

func (w *ConsolidatedWorkloadSnapshotModel) Create(ctx context.Context) error {
	q := `INSERT INTO consolidated_workload_snapshots (
		consolidation_id, namespace, pod, container, cpu_requested_cores, cpu_used_cores, cpu_limit_cores,
		memory_requested_bytes, memory_used_bytes, memory_limit_bytes, cpu_waste_ratio, mem_waste_ratio,
		projected_monthly_waste_usd
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING id`
	return DefaultDB.QueryRowContext(ctx, q,
		w.consolidationID, w.namespace, w.pod, w.container, w.cpuRequestedCores, w.cpuUsedCores, w.cpuLimitCores,
		w.memoryRequestedBytes, w.memoryUsedBytes, w.memoryLimitBytes, w.cpuWasteRatio, w.memWasteRatio,
		w.projectedMonthlyWasteUSD).Scan(&w.id)
}

func (w *ConsolidatedWorkloadSnapshotModel) Update(ctx context.Context) error {
	q := `UPDATE consolidated_workload_snapshots SET
		consolidation_id = $1, namespace = $2, pod = $3, container = $4, cpu_requested_cores = $5, cpu_used_cores = $6,
		cpu_limit_cores = $7, memory_requested_bytes = $8, memory_used_bytes = $9, memory_limit_bytes = $10,
		cpu_waste_ratio = $11, mem_waste_ratio = $12, projected_monthly_waste_usd = $13
		WHERE id = $14`
	_, err := DefaultDB.ExecContext(ctx, q,
		w.consolidationID, w.namespace, w.pod, w.container, w.cpuRequestedCores, w.cpuUsedCores, w.cpuLimitCores,
		w.memoryRequestedBytes, w.memoryUsedBytes, w.memoryLimitBytes, w.cpuWasteRatio, w.memWasteRatio,
		w.projectedMonthlyWasteUSD, w.id)
	return err
}

func (w *ConsolidatedWorkloadSnapshotModel) Delete(ctx context.Context) error {
	q := `DELETE FROM consolidated_workload_snapshots WHERE id = $1`
	_, err := DefaultDB.ExecContext(ctx, q, w.id)
	return err
}

func (w *ConsolidatedWorkloadSnapshotModel) Read(ctx context.Context, filters map[string]any) error {
	where, args := buildWhereClause(filters)
	q := `SELECT id, consolidation_id, namespace, pod, container, cpu_requested_cores, cpu_used_cores, cpu_limit_cores,
		memory_requested_bytes, memory_used_bytes, memory_limit_bytes, cpu_waste_ratio, mem_waste_ratio,
		projected_monthly_waste_usd FROM consolidated_workload_snapshots` + where + ` LIMIT 1`
	return DefaultDB.QueryRowContext(ctx, q, args...).Scan(
		&w.id, &w.consolidationID, &w.namespace, &w.pod, &w.container, &w.cpuRequestedCores, &w.cpuUsedCores, &w.cpuLimitCores,
		&w.memoryRequestedBytes, &w.memoryUsedBytes, &w.memoryLimitBytes, &w.cpuWasteRatio, &w.memWasteRatio,
		&w.projectedMonthlyWasteUSD)
}

func (w *ConsolidatedWorkloadSnapshotModel) ReadAll(ctx context.Context, filters map[string]any) ([]Entity, error) {
	where, args := buildWhereClause(filters)
	q := `SELECT id, consolidation_id, namespace, pod, container, cpu_requested_cores, cpu_used_cores, cpu_limit_cores,
		memory_requested_bytes, memory_used_bytes, memory_limit_bytes, cpu_waste_ratio, mem_waste_ratio,
		projected_monthly_waste_usd FROM consolidated_workload_snapshots` + where
	rows, err := DefaultDB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Entity
	for rows.Next() {
		item := &ConsolidatedWorkloadSnapshotModel{}
		err := rows.Scan(
			&item.id, &item.consolidationID, &item.namespace, &item.pod, &item.container, &item.cpuRequestedCores, &item.cpuUsedCores, &item.cpuLimitCores,
			&item.memoryRequestedBytes, &item.memoryUsedBytes, &item.memoryLimitBytes, &item.cpuWasteRatio, &item.memWasteRatio,
			&item.projectedMonthlyWasteUSD)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

// Static helper mapping
type consolidatedWorkloadSnapshotStatic struct{}

var ConsolidatedWorkloadSnapshot consolidatedWorkloadSnapshotStatic

func (consolidatedWorkloadSnapshotStatic) Migrate(ctx context.Context) error {
	m := &ConsolidatedWorkloadSnapshotModel{}
	return m.Migrate(ctx)
}

func (consolidatedWorkloadSnapshotStatic) Create(ctx context.Context, w *ConsolidatedWorkloadSnapshotModel) error {
	return w.Create(ctx)
}

func (consolidatedWorkloadSnapshotStatic) Update(ctx context.Context, w *ConsolidatedWorkloadSnapshotModel) error {
	return w.Update(ctx)
}

func (consolidatedWorkloadSnapshotStatic) Delete(ctx context.Context, w *ConsolidatedWorkloadSnapshotModel) error {
	return w.Delete(ctx)
}

func (consolidatedWorkloadSnapshotStatic) Read(ctx context.Context, w *ConsolidatedWorkloadSnapshotModel, filters map[string]any) error {
	return w.Read(ctx, filters)
}

func (consolidatedWorkloadSnapshotStatic) ReadAll(ctx context.Context, filters map[string]any) ([]*ConsolidatedWorkloadSnapshotModel, error) {
	m := &ConsolidatedWorkloadSnapshotModel{}
	entities, err := m.ReadAll(ctx, filters)
	if err != nil {
		return nil, err
	}
	res := make([]*ConsolidatedWorkloadSnapshotModel, len(entities))
	for i, e := range entities {
		res[i] = e.(*ConsolidatedWorkloadSnapshotModel)
	}
	return res, nil
}
