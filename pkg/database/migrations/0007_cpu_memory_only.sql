TRUNCATE TABLE scrapes, consolidations CASCADE;

ALTER TABLE indicator_snapshots
    DROP COLUMN IF EXISTS pvc_waste_ratio,
    DROP COLUMN IF EXISTS hpa_efficiency,
    DROP COLUMN IF EXISTS hpa_avg_replicas,
    DROP COLUMN IF EXISTS hpa_max_replicas,
    DROP COLUMN IF EXISTS pvc_capacity_bytes,
    DROP COLUMN IF EXISTS pvc_used_bytes,
    DROP COLUMN IF EXISTS oom_risk_score,
    ADD COLUMN IF NOT EXISTS cpu_projected_monthly_waste_usd NUMERIC(12,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS memory_projected_monthly_waste_usd NUMERIC(12,4) NOT NULL DEFAULT 0;

ALTER TABLE workload_item_snapshots
    DROP COLUMN IF EXISTS oom_risk_score,
    ADD COLUMN IF NOT EXISTS cpu_projected_monthly_waste_usd NUMERIC(12,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS memory_projected_monthly_waste_usd NUMERIC(12,4) NOT NULL DEFAULT 0;

ALTER TABLE consolidations
    DROP COLUMN IF EXISTS pvc_waste_ratio,
    DROP COLUMN IF EXISTS hpa_efficiency,
    DROP COLUMN IF EXISTS oom_risk_score,
    ADD COLUMN IF NOT EXISTS cpu_projected_monthly_waste_usd NUMERIC(12,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS memory_projected_monthly_waste_usd NUMERIC(12,4) NOT NULL DEFAULT 0;

ALTER TABLE consolidated_workload_snapshots
    DROP COLUMN IF EXISTS oom_risk_score,
    ADD COLUMN IF NOT EXISTS cpu_projected_monthly_waste_usd NUMERIC(12,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS memory_projected_monthly_waste_usd NUMERIC(12,4) NOT NULL DEFAULT 0;
