CREATE TABLE plane_go_runtime_probes (
    name text PRIMARY KEY,
    checked_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);
