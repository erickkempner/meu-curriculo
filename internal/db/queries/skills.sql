-- name: CreateSkill :one
INSERT INTO skills (resume_id, name, display_order)
VALUES ($1, $2, $3)
RETURNING *;

-- name: FindSkillsByResumeID :many
SELECT * FROM skills WHERE resume_id = $1 ORDER BY display_order;

-- name: DeleteSkillsByResumeID :exec
DELETE FROM skills WHERE resume_id = $1;
