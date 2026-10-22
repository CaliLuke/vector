-- name: PutUUIDProbe :one
INSERT INTO plane_go_uuid_probes (id, nullable_id, related_ids)
VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE
SET nullable_id = EXCLUDED.nullable_id,
    related_ids = EXCLUDED.related_ids
RETURNING id, nullable_id, related_ids;

-- name: GetUUIDProbe :one
SELECT id, nullable_id, related_ids
FROM plane_go_uuid_probes
WHERE id = $1;
