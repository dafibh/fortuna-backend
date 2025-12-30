-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByAuth0ID :one
SELECT * FROM users WHERE auth0_id = $1;

-- name: CreateUser :one
INSERT INTO users (auth0_id, email, name, picture_url)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET email = $2, name = $3, picture_url = $4, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CreateOrGetUserByAuth0ID :one
INSERT INTO users (auth0_id, email, name, picture_url)
VALUES ($1, $2, $3, $4)
ON CONFLICT (auth0_id) DO UPDATE SET
    email = EXCLUDED.email,
    name = EXCLUDED.name,
    picture_url = EXCLUDED.picture_url,
    updated_at = NOW()
RETURNING *;

-- name: UpdateUserName :one
UPDATE users
SET name = $2, updated_at = NOW()
WHERE auth0_id = $1
RETURNING *;
