-- name: CreateExperience :one
INSERT INTO experiences (resume_id, company, role, period, description, display_order)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: FindExperiencesByResumeID :many
SELECT * FROM experiences WHERE resume_id = $1 ORDER BY display_order;

-- name: DeleteExperiencesByResumeID :exec
DELETE FROM experiences WHERE resume_id = $1;
