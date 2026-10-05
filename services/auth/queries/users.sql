-- name: CreateUser :one
INSERT INTO users (
  email,
  password_hash,
  full_name,
  persona,
  timezone,
  work_start,
  work_end
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: UpdateUserProfile :one
UPDATE users
SET
  full_name = $2,
  persona = $3,
  timezone = $4,
  work_start = $5,
  work_end = $6,
  updated_at = now()
WHERE id = $1
RETURNING *;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
  user_id,
  token_hash,
  expires_at
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token_hash = $1 LIMIT 1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked = true
WHERE id = $1;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens
SET revoked = true
WHERE user_id = $1;

-- name: ListBriefingCandidates :many
SELECT id, email, full_name, persona, timezone, work_start, work_end
FROM users;
