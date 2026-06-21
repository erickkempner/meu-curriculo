-- name: CreateUser :one
INSERT INTO users (name, email, password_hash, provider)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: FindUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: FindUserByID :one
SELECT * FROM users WHERE id = $1;
