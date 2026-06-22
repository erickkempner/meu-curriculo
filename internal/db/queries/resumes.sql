-- name: CreateResume :one
INSERT INTO resumes (user_id, title, template_name, personal_name, personal_title, email, phone, location, summary, photo_url)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: FindResumeByID :one
SELECT * FROM resumes WHERE id = $1;

-- name: FindResumesByUserID :many
SELECT * FROM resumes WHERE user_id = $1 ORDER BY updated_at DESC;

-- name: UpdateResume :exec
UPDATE resumes SET title=$2, template_name=$3, personal_name=$4, personal_title=$5,
    email=$6, phone=$7, location=$8, summary=$9, photo_url=$10, updated_at=NOW()
WHERE id = $1;

-- name: DeleteResume :exec
DELETE FROM resumes WHERE id = $1;

-- name: FindResumeByShareToken :one
SELECT * FROM resumes WHERE share_token = $1;

-- name: SetShareToken :exec
UPDATE resumes SET share_token = $2 WHERE id = $1;

-- name: UpdatePhotoURL :exec
UPDATE resumes SET photo_url = $2, updated_at = NOW() WHERE id = $1;
