CREATE TABLE IF NOT EXISTS indicator_snapshots (
    id SERIAL PRIMARY KEY,
    scrape_id INT REFERENCES scrapes(id) ON DELETE CASCADE,
    cpu_waste_ratio NUMERIC(5,4) NOT NULL,
    mem_waste_ratio NUMERIC(5,4) NOT NULL,
    pvc_waste_ratio NUMERIC(5,4) NOT NULL,
    hpa_efficiency NUMERIC(5,4) NOT NULL,
    projected_monthly_waste_usd NUMERIC(12,4) NOT NULL,
    total_cpu_requested NUMERIC(12,4) NOT NULL,
    total_cpu_used NUMERIC(12,4) NOT NULL,
    total_memory_requested_bytes NUMERIC(20,4) NOT NULL,
    total_memory_used_bytes NUMERIC(20,4) NOT NULL,
    hpa_avg_replicas NUMERIC(8,2) NOT NULL,
    hpa_max_replicas NUMERIC(8,2) NOT NULL,
    pvc_capacity_bytes NUMERIC(20,4) NOT NULL,
    pvc_used_bytes NUMERIC(20,4) NOT NULL
);
