-- name: CreateProjectionExclusion :exec
INSERT INTO projection_exclusions (workspace_id, template_id, excluded_month)
VALUES ($1, $2, $3)
ON CONFLICT (workspace_id, template_id, excluded_month) DO NOTHING;

-- name: IsMonthExcluded :one
SELECT EXISTS(
    SELECT 1 FROM projection_exclusions
    WHERE workspace_id = $1 AND template_id = $2 AND excluded_month = $3
) AS excluded;

-- name: DeleteExclusionsByTemplate :exec
DELETE FROM projection_exclusions WHERE template_id = $1;

-- name: GetExclusionsByTemplate :many
SELECT * FROM projection_exclusions
WHERE workspace_id = $1 AND template_id = $2
ORDER BY excluded_month;
