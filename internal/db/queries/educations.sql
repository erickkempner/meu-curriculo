-- name: CreateEducation :one
INSERT INTO educations (resume_id, institution, degree, period, display_order)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: FindEducationsByResumeID :many
SELECT * FROM educations WHERE resume_id = $1 ORDER BY display_order;

-- name: DeleteEducationsByResumeID :exec
DELETE FROM educations WHERE resume_id = $1;
