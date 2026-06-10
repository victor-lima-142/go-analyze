package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"go-analyze/internal/metrics"
	"go-analyze/pkg/config"
	"go-analyze/pkg/database/entities"
	"go-analyze/pkg/database/migrations"

	"github.com/lib/pq"
)

type Database struct {
	db *sql.DB
}

func Connect(cfg *config.Config) (*Database, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	entities.DefaultDB = db

	return &Database{db: db}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

// Migrate delegates schema management to the versioned migrations package.
func (d *Database) Migrate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return migrations.Run(ctx, d.db)
}

// HealthCheck verifies that the database is reachable and the schema is
// initialised. It is safe to be called from /healthz handlers.
func (d *Database) HealthCheck(ctx context.Context) error {
	if err := d.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping: %w", err)
	}
	var n int
	if err := d.db.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&n); err != nil {
		return fmt.Errorf("schema_migrations missing: %w", err)
	}
	return nil
}

// SaveSnapshot persists a scrape, the global indicator snapshot, and all
// per-workload items inside a single transaction so failures cannot leave the
// history in an inconsistent state.
func (d *Database) SaveSnapshot(ctx context.Context, res *metrics.IndicatorResponse, duration time.Duration) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var scrapeID int
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO scrapes (scraped_at, duration_ms) VALUES ($1, $2) RETURNING id`,
		time.Now(), int(duration.Milliseconds()),
	).Scan(&scrapeID); err != nil {
		return fmt.Errorf("insert scrape: %w", err)
	}

	ind := res.Indicators
	inp := res.Inputs
	if _, err := tx.ExecContext(ctx, `INSERT INTO indicator_snapshots (
        scrape_id, cpu_waste_ratio, mem_waste_ratio, pvc_waste_ratio, hpa_efficiency, projected_monthly_waste_usd,
        total_cpu_requested, total_cpu_used, total_memory_requested_bytes, total_memory_used_bytes,
        hpa_avg_replicas, hpa_max_replicas, pvc_capacity_bytes, pvc_used_bytes, oom_risk_score
    ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		scrapeID,
		ind.CPUWasteRatio, ind.MemWasteRatio, ind.PVCWasteRatio, ind.HPAEfficiency, ind.ProjectedMonthlyWasteUSD,
		inp.CPURequestedCores, inp.CPUUsedCores, inp.MemoryRequestedBytes, inp.MemoryUsedBytes,
		inp.HPAAvgReplicas, inp.HPAMaxReplicas, inp.PVCCapacityBytes, inp.PVCUsedBytes,
		ind.OOMRiskScore,
	); err != nil {
		return fmt.Errorf("insert indicator_snapshot: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO workload_item_snapshots (
        scrape_id, namespace, pod, container, cpu_requested_cores, cpu_used_cores, cpu_limit_cores,
        memory_requested_bytes, memory_used_bytes, memory_limit_bytes, cpu_waste_ratio, mem_waste_ratio,
        projected_monthly_waste_usd, oom_risk_score
    ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`)
	if err != nil {
		return fmt.Errorf("prepare workload insert: %w", err)
	}
	defer stmt.Close()

	for _, item := range res.Items {
		cpuLimit := nullableFloat(item.CPULimitCores)
		memLimit := nullableFloat(item.MemoryLimitBytes)
		if _, err := stmt.ExecContext(ctx,
			scrapeID, item.Namespace, item.Pod, item.Container, item.CPURequestedCores, item.CPUUsedCores, cpuLimit,
			item.MemoryRequestedBytes, item.MemoryUsedBytes, memLimit, item.CPUWasteRatio, item.MemWasteRatio,
			item.ProjectedMonthlyWasteUSD, item.OOMRiskScore,
		); err != nil {
			return fmt.Errorf("insert workload item: %w", err)
		}
	}

	return tx.Commit()
}

func nullableFloat(v float64) any {
	if v > 0 {
		return v
	}
	return nil
}

func (d *Database) GetScrapesInRange(ctx context.Context, start, end time.Time) ([]*entities.ScrapeModel, error) {
	q := `SELECT id, scraped_at, duration_ms FROM scrapes WHERE scraped_at >= $1 AND scraped_at <= $2 ORDER BY scraped_at ASC`
	rows, err := d.db.QueryContext(ctx, q, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scrapes []*entities.ScrapeModel
	for rows.Next() {
		s := &entities.ScrapeModel{}
		var scrapedAt time.Time
		var id, durationMs int
		if err := rows.Scan(&id, &scrapedAt, &durationMs); err != nil {
			return nil, err
		}
		s.SetID(id)
		s.SetScrapedAt(scrapedAt)
		s.SetDurationMs(durationMs)
		scrapes = append(scrapes, s)
	}
	return scrapes, nil
}

func (d *Database) GetIndicatorSnapshotsForScrapes(ctx context.Context, scrapeIDs []int) ([]*entities.IndicatorSnapshotModel, error) {
	if len(scrapeIDs) == 0 {
		return nil, nil
	}
	q := `SELECT id, scrape_id, cpu_waste_ratio, mem_waste_ratio, pvc_waste_ratio, hpa_efficiency, projected_monthly_waste_usd,
		total_cpu_requested, total_cpu_used, total_memory_requested_bytes, total_memory_used_bytes,
		hpa_avg_replicas, hpa_max_replicas, pvc_capacity_bytes, pvc_used_bytes
		FROM indicator_snapshots WHERE scrape_id = ANY($1)`
	rows, err := d.db.QueryContext(ctx, q, pq.Array(scrapeIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*entities.IndicatorSnapshotModel
	for rows.Next() {
		var id, scrapeID int
		var cpuWaste, memWaste, pvcWaste, hpaEff, projected float64
		var totalCPUReq, totalCPUUsed, totalMemReq, totalMemUsed float64
		var hpaAvg, hpaMax, pvcCap, pvcUsed float64
		err := rows.Scan(
			&id, &scrapeID, &cpuWaste, &memWaste, &pvcWaste, &hpaEff, &projected,
			&totalCPUReq, &totalCPUUsed, &totalMemReq, &totalMemUsed,
			&hpaAvg, &hpaMax, &pvcCap, &pvcUsed,
		)
		if err != nil {
			return nil, err
		}
		item := entities.NewIndicatorSnapshot(
			scrapeID, cpuWaste, memWaste, pvcWaste, hpaEff, projected,
			totalCPUReq, totalCPUUsed, totalMemReq, totalMemUsed,
			hpaAvg, hpaMax, pvcCap, pvcUsed,
		)
		item.SetID(id)
		list = append(list, item)
	}
	return list, nil
}

func (d *Database) GetWorkloadItemSnapshotsForScrapes(ctx context.Context, scrapeIDs []int) ([]*entities.WorkloadItemSnapshotModel, error) {
	if len(scrapeIDs) == 0 {
		return nil, nil
	}
	q := `SELECT id, scrape_id, namespace, pod, container, cpu_requested_cores, cpu_used_cores, cpu_limit_cores,
		memory_requested_bytes, memory_used_bytes, memory_limit_bytes, cpu_waste_ratio, mem_waste_ratio,
		projected_monthly_waste_usd
		FROM workload_item_snapshots WHERE scrape_id = ANY($1)`
	rows, err := d.db.QueryContext(ctx, q, pq.Array(scrapeIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*entities.WorkloadItemSnapshotModel
	for rows.Next() {
		var id, scrapeID int
		var ns, pod, container string
		var cpuReq, cpuUsed float64
		var cpuLimit *float64
		var memReq, memUsed float64
		var memLimit *float64
		var cpuWaste, memWaste, projected float64
		err := rows.Scan(
			&id, &scrapeID, &ns, &pod, &container, &cpuReq, &cpuUsed, &cpuLimit,
			&memReq, &memUsed, &memLimit, &cpuWaste, &memWaste, &projected,
		)
		if err != nil {
			return nil, err
		}
		item := entities.NewWorkloadItemSnapshot(
			scrapeID, ns, pod, container, cpuReq, cpuUsed, cpuLimit,
			memReq, memUsed, memLimit, cpuWaste, memWaste, projected,
		)
		item.SetID(id)
		list = append(list, item)
	}
	return list, nil
}

func (d *Database) SaveConsolidation(ctx context.Context, consolidation *entities.ConsolidationModel, workloads []*entities.ConsolidatedWorkloadSnapshotModel) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	q1 := `INSERT INTO consolidations (
		consolidated_at, start_time, end_time, scrapes_count, cpu_waste_ratio, mem_waste_ratio,
		pvc_waste_ratio, hpa_efficiency, projected_monthly_waste_usd, total_cpu_requested, total_cpu_used,
		total_memory_requested_bytes, total_memory_used_bytes, oom_risk_score
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) RETURNING id`
	var cid int
	err = tx.QueryRowContext(ctx, q1,
		consolidation.ConsolidatedAt(), consolidation.StartTime(), consolidation.EndTime(),
		consolidation.ScrapesCount(), consolidation.CPUWasteRatio(), consolidation.MemWasteRatio(),
		consolidation.PVCWasteRatio(), consolidation.HPAEfficiency(), consolidation.ProjectedMonthlyWasteUSD(),
		consolidation.TotalCPURequested(), consolidation.TotalCPUUsed(),
		consolidation.TotalMemoryRequested(), consolidation.TotalMemoryUsed(),
		consolidation.OOMRiskScore(),
	).Scan(&cid)
	if err != nil {
		return fmt.Errorf("failed to save consolidation header in transaction: %w", err)
	}
	consolidation.SetID(cid)

	q2 := `INSERT INTO consolidated_workload_snapshots (
		consolidation_id, namespace, pod, container, cpu_requested_cores, cpu_used_cores, cpu_limit_cores,
		memory_requested_bytes, memory_used_bytes, memory_limit_bytes, cpu_waste_ratio, mem_waste_ratio,
		projected_monthly_waste_usd, oom_risk_score
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`
	stmt, err := tx.PrepareContext(ctx, q2)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, w := range workloads {
		_, err = stmt.ExecContext(ctx,
			cid, w.Namespace(), w.Pod(), w.Container(), w.CPURequestedCores(), w.CPUUsedCores(), w.CPULimitCores(),
			w.MemoryRequestedBytes(), w.MemoryUsedBytes(), w.MemoryLimitBytes(), w.CPUWasteRatio(), w.MemWasteRatio(),
			w.ProjectedMonthlyWasteUSD(), w.OOMRiskScore())
		if err != nil {
			return fmt.Errorf("failed to save consolidated workload snapshot: %w", err)
		}
	}

	return tx.Commit()
}

func (d *Database) GetConsolidations(ctx context.Context, limit, offset int, startTime, endTime *time.Time) ([]*entities.ConsolidationModel, int, error) {
	var countQuery strings.Builder
	countQuery.WriteString(`SELECT COUNT(*) FROM consolidations`)

	var selectQuery strings.Builder
	selectQuery.WriteString(`SELECT id, consolidated_at, start_time, end_time, scrapes_count, cpu_waste_ratio, mem_waste_ratio,
		pvc_waste_ratio, hpa_efficiency, projected_monthly_waste_usd, total_cpu_requested, total_cpu_used,
		total_memory_requested_bytes, total_memory_used_bytes
		FROM consolidations`)

	var whereParts []string
	var args []any
	paramID := 1

	if startTime != nil {
		whereParts = append(whereParts, fmt.Sprintf("consolidated_at >= $%d", paramID))
		args = append(args, *startTime)
		paramID++
	}
	if endTime != nil {
		whereParts = append(whereParts, fmt.Sprintf("consolidated_at <= $%d", paramID))
		args = append(args, *endTime)
		paramID++
	}

	whereClause := ""
	if len(whereParts) > 0 {
		whereClause = " WHERE " + strings.Join(whereParts, " AND ")
		countQuery.WriteString(whereClause)
		selectQuery.WriteString(whereClause)
	}

	var totalItems int
	err := d.db.QueryRowContext(ctx, countQuery.String(), args...).Scan(&totalItems)
	if err != nil {
		return nil, 0, err
	}

	selectQuery.WriteString(" ORDER BY consolidated_at DESC")
	selectQuery.WriteString(fmt.Sprintf(" LIMIT $%d OFFSET $%d", paramID, paramID+1))
	args = append(args, limit, offset)

	rows, err := d.db.QueryContext(ctx, selectQuery.String(), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*entities.ConsolidationModel
	for rows.Next() {
		var id, scrapesCount int
		var consolidatedAt, startTimeVal, endTimeVal time.Time
		var cpuWaste, memWaste, pvcWaste, hpaEff, projected float64
		var totalCPUReq, totalCPUUsed, totalMemReq, totalMemUsed float64
		err := rows.Scan(
			&id, &consolidatedAt, &startTimeVal, &endTimeVal, &scrapesCount, &cpuWaste, &memWaste,
			&pvcWaste, &hpaEff, &projected, &totalCPUReq, &totalCPUUsed,
			&totalMemReq, &totalMemUsed,
		)
		if err != nil {
			return nil, 0, err
		}
		item := entities.NewConsolidation(
			startTimeVal, endTimeVal, scrapesCount, cpuWaste, memWaste, pvcWaste, hpaEff, projected,
			totalCPUReq, totalCPUUsed, totalMemReq, totalMemUsed,
		)
		item.SetID(id)
		item.SetConsolidatedAt(consolidatedAt)
		list = append(list, item)
	}
	return list, totalItems, nil
}

func (d *Database) GetConsolidatedWorkloads(ctx context.Context, consolidationID int) ([]*entities.ConsolidatedWorkloadSnapshotModel, error) {
	q := `SELECT id, consolidation_id, namespace, pod, container, cpu_requested_cores, cpu_used_cores, cpu_limit_cores,
		memory_requested_bytes, memory_used_bytes, memory_limit_bytes, cpu_waste_ratio, mem_waste_ratio,
		projected_monthly_waste_usd
		FROM consolidated_workload_snapshots WHERE consolidation_id = $1`
	rows, err := d.db.QueryContext(ctx, q, consolidationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*entities.ConsolidatedWorkloadSnapshotModel
	for rows.Next() {
		var id, cid int
		var ns, pod, container string
		var cpuReq, cpuUsed float64
		var cpuLimit *float64
		var memReq, memUsed float64
		var memLimit *float64
		var cpuWaste, memWaste, projected float64
		err := rows.Scan(
			&id, &cid, &ns, &pod, &container, &cpuReq, &cpuUsed, &cpuLimit,
			&memReq, &memUsed, &memLimit, &cpuWaste, &memWaste, &projected,
		)
		if err != nil {
			return nil, err
		}
		item := entities.NewConsolidatedWorkloadSnapshot(
			cid, ns, pod, container, cpuReq, cpuUsed, cpuLimit,
			memReq, memUsed, memLimit, cpuWaste, memWaste, projected,
		)
		item.SetID(id)
		list = append(list, item)
	}
	return list, nil
}

// GetLastWorkloadSnapshots returns historical consolidated snapshots for the
// (namespace, container) pair. The pod argument behaves as a prefix matcher —
// callers may pass an explicit "%" wildcard for finer control.
func (d *Database) GetLastWorkloadSnapshots(ctx context.Context, namespace, pod, container string, limit int, startTime, endTime *time.Time) ([]*entities.HistoricalWorkloadSnapshot, error) {
	var q strings.Builder
	q.WriteString(`SELECT w.id, w.consolidation_id, w.namespace, w.pod, w.container, w.cpu_requested_cores, w.cpu_used_cores, w.cpu_limit_cores,
		w.memory_requested_bytes, w.memory_used_bytes, w.memory_limit_bytes, w.cpu_waste_ratio, w.mem_waste_ratio, w.projected_monthly_waste_usd,
		c.consolidated_at
		FROM consolidated_workload_snapshots w
		JOIN consolidations c ON w.consolidation_id = c.id
		WHERE w.namespace = $1 AND w.container = $2`)

	args := []any{namespace, container}
	paramID := 3

	if pod != "" {
		podParam := pod
		if !strings.Contains(pod, "%") {
			podParam = pod + "%"
		}
		q.WriteString(fmt.Sprintf(" AND w.pod LIKE $%d", paramID))
		args = append(args, podParam)
		paramID++
	}

	if startTime != nil {
		q.WriteString(fmt.Sprintf(" AND c.consolidated_at >= $%d", paramID))
		args = append(args, *startTime)
		paramID++
	}

	if endTime != nil {
		q.WriteString(fmt.Sprintf(" AND c.consolidated_at <= $%d", paramID))
		args = append(args, *endTime)
		paramID++
	}

	q.WriteString(" ORDER BY c.consolidated_at DESC")

	if limit > 0 {
		q.WriteString(fmt.Sprintf(" LIMIT $%d", paramID))
		args = append(args, limit)
	}

	rows, err := d.db.QueryContext(ctx, q.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*entities.HistoricalWorkloadSnapshot
	for rows.Next() {
		var id, cid int
		var ns, p, containerName string
		var cpuReq, cpuUsed float64
		var cpuLimit *float64
		var memReq, memUsed float64
		var memLimit *float64
		var cpuWaste, memWaste, projected float64
		var consolidatedAt time.Time

		err := rows.Scan(
			&id, &cid, &ns, &p, &containerName, &cpuReq, &cpuUsed, &cpuLimit,
			&memReq, &memUsed, &memLimit, &cpuWaste, &memWaste, &projected,
			&consolidatedAt,
		)
		if err != nil {
			return nil, err
		}

		snap := entities.NewConsolidatedWorkloadSnapshot(
			cid, ns, p, containerName, cpuReq, cpuUsed, cpuLimit,
			memReq, memUsed, memLimit, cpuWaste, memWaste, projected,
		)
		snap.SetID(id)

		list = append(list, &entities.HistoricalWorkloadSnapshot{
			Snapshot:       snap,
			ConsolidatedAt: consolidatedAt,
		})
	}
	return list, nil
}
