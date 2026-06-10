CREATE TABLE IF NOT EXISTS consolidated_workload_snapshots (
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
);

CREATE INDEX IF NOT EXISTS idx_consolidated_workload_lookup ON consolidated_workload_snapshots(namespace, pod, container);
