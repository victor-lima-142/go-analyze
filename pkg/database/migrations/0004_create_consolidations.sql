CREATE TABLE IF NOT EXISTS consolidations (
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
);

CREATE INDEX IF NOT EXISTS idx_consolidations_consolidated_at ON consolidations(consolidated_at);
