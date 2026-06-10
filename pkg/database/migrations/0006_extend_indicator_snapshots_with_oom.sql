ALTER TABLE indicator_snapshots
    ADD COLUMN IF NOT EXISTS oom_risk_score NUMERIC(5,4) NOT NULL DEFAULT 0;

ALTER TABLE workload_item_snapshots
    ADD COLUMN IF NOT EXISTS oom_risk_score NUMERIC(5,4) NOT NULL DEFAULT 0;

ALTER TABLE consolidated_workload_snapshots
    ADD COLUMN IF NOT EXISTS oom_risk_score NUMERIC(5,4) NOT NULL DEFAULT 0;

ALTER TABLE consolidations
    ADD COLUMN IF NOT EXISTS oom_risk_score NUMERIC(5,4) NOT NULL DEFAULT 0;
