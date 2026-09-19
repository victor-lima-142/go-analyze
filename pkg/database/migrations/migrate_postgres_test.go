package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func integrationDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	schema := fmt.Sprintf("go_analyze_migration_%d", time.Now().UnixNano())
	if _, err = db.Exec(`CREATE SCHEMA "` + schema + `"`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if _, err = db.Exec(`SET search_path TO "` + schema + `"`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DROP SCHEMA IF EXISTS "` + schema + `" CASCADE`); _ = db.Close() })
	return db, schema
}

func TestPostgresFreshUpgradePersistenceTransactionAndCascade(t *testing.T) {
	ctx := context.Background()
	t.Run("fresh", func(t *testing.T) {
		db, schema := integrationDB(t)
		if err := Run(ctx, db); err != nil {
			t.Fatal(err)
		}
		assertFinalColumns(t, db, schema)
		var scrapeID int
		if err := db.QueryRow(`INSERT INTO scrapes(duration_ms) VALUES(1) RETURNING id`).Scan(&scrapeID); err != nil {
			t.Fatal(err)
		}
		_, err := db.Exec(`INSERT INTO indicator_snapshots(scrape_id,cpu_waste_ratio,mem_waste_ratio,cpu_projected_monthly_waste_usd,memory_projected_monthly_waste_usd,projected_monthly_waste_usd,total_cpu_requested,total_cpu_used,total_memory_requested_bytes,total_memory_used_bytes) VALUES($1,.5,.25,3,2,5,2,1,2048,1024)`, scrapeID)
		if err != nil {
			t.Fatal(err)
		}
		var cpu, memory, total float64
		if err := db.QueryRow(`SELECT cpu_projected_monthly_waste_usd,memory_projected_monthly_waste_usd,projected_monthly_waste_usd FROM indicator_snapshots WHERE scrape_id=$1`, scrapeID).Scan(&cpu, &memory, &total); err != nil {
			t.Fatal(err)
		}
		if cpu+memory != total {
			t.Fatalf("cost decomposition %v + %v != %v", cpu, memory, total)
		}
		tx, _ := db.Begin()
		_, _ = tx.Exec(`INSERT INTO scrapes(duration_ms) VALUES(99)`)
		_ = tx.Rollback()
		var rolledBack int
		_ = db.QueryRow(`SELECT count(*) FROM scrapes WHERE duration_ms=99`).Scan(&rolledBack)
		if rolledBack != 0 {
			t.Fatal("transaction did not roll back")
		}
		if _, err := db.Exec(`DELETE FROM scrapes WHERE id=$1`, scrapeID); err != nil {
			t.Fatal(err)
		}
		var children int
		_ = db.QueryRow(`SELECT count(*) FROM indicator_snapshots WHERE scrape_id=$1`, scrapeID).Scan(&children)
		if children != 0 {
			t.Fatal("cascade did not delete snapshot")
		}
	})

	t.Run("upgrade", func(t *testing.T) {
		db, schema := integrationDB(t)
		if _, err := db.Exec(schemaTable); err != nil {
			t.Fatal(err)
		}
		all, err := loadAll()
		if err != nil {
			t.Fatal(err)
		}
		for _, migration := range all[:len(all)-1] {
			if err := applyOne(ctx, db, migration); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := db.Exec(`INSERT INTO scrapes(duration_ms) VALUES(1)`); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO consolidations(start_time,end_time,scrapes_count,cpu_waste_ratio,mem_waste_ratio,pvc_waste_ratio,hpa_efficiency,projected_monthly_waste_usd,total_cpu_requested,total_cpu_used,total_memory_requested_bytes,total_memory_used_bytes) VALUES(now(),now(),1,0,0,0,0,0,0,0,0,0)`); err != nil {
			t.Fatal(err)
		}
		if err := Run(ctx, db); err != nil {
			t.Fatal(err)
		}
		assertFinalColumns(t, db, schema)
		for _, table := range []string{"scrapes", "consolidations"} {
			var count int
			if err := db.QueryRow(`SELECT count(*) FROM ` + table).Scan(&count); err != nil || count != 0 {
				t.Fatalf("%s not cleaned: count=%d err=%v", table, count, err)
			}
		}
	})
}

func assertFinalColumns(t *testing.T, db *sql.DB, schema string) {
	t.Helper()
	for _, column := range []string{"cpu_projected_monthly_waste_usd", "memory_projected_monthly_waste_usd"} {
		var count int
		if err := db.QueryRow(`SELECT count(*) FROM information_schema.columns WHERE table_schema=$1 AND table_name='indicator_snapshots' AND column_name=$2`, schema, column).Scan(&count); err != nil || count != 1 {
			t.Fatalf("missing %s: %v", column, err)
		}
	}
	for _, column := range []string{"pvc_waste_ratio", "hpa_efficiency", "oom_risk_score"} {
		var count int
		_ = db.QueryRow(`SELECT count(*) FROM information_schema.columns WHERE table_schema=$1 AND column_name=$2`, schema, column).Scan(&count)
		if count != 0 {
			t.Fatalf("legacy column remains: %s", column)
		}
	}
}
