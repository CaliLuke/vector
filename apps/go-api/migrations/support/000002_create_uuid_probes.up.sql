CREATE TABLE plane_go_uuid_probes (
    id uuid PRIMARY KEY,
    nullable_id uuid,
    related_ids uuid[] NOT NULL
);
