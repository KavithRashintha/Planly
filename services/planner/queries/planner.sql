-- name: CreateProject :one
INSERT INTO projects (
  user_id,
  name,
  colour
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: GetProjectByID :one
SELECT * FROM projects
WHERE id = $1 AND user_id = $2
LIMIT 1;

-- name: ListProjects :many
SELECT * FROM projects
WHERE user_id = $1
ORDER BY name ASC;

-- name: UpdateProject :one
UPDATE projects
SET
  name = $3,
  colour = $4,
  archived = $5,
  updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteProject :exec
DELETE FROM projects
WHERE id = $1 AND user_id = $2;

-- name: CreateTask :one
INSERT INTO tasks (
  user_id,
  project_id,
  title,
  description,
  priority,
  status,
  due_at,
  estimate_minutes,
  completed_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: GetTaskByID :one
SELECT * FROM tasks
WHERE id = $1 AND user_id = $2
LIMIT 1;

-- name: UpdateTask :one
UPDATE tasks
SET
  project_id = $3,
  title = $4,
  description = $5,
  priority = $6,
  status = $7,
  due_at = $8,
  estimate_minutes = $9,
  completed_at = $10,
  updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteTask :exec
DELETE FROM tasks
WHERE id = $1 AND user_id = $2;

-- name: CreateSubtask :one
INSERT INTO subtasks (
  task_id,
  title,
  position
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: ListSubtasksByTaskID :many
SELECT s.* FROM subtasks s
JOIN tasks t ON s.task_id = t.id
WHERE t.id = $1 AND t.user_id = $2
ORDER BY s.position ASC, s.id ASC;

-- name: UpdateSubtask :one
UPDATE subtasks s
SET
  title = $3,
  done = $4,
  position = $5
FROM tasks t
WHERE s.task_id = t.id AND s.id = $1 AND t.user_id = $2
RETURNING s.*;

-- name: DeleteSubtask :exec
DELETE FROM subtasks s
USING tasks t
WHERE s.task_id = t.id AND s.id = $1 AND t.user_id = $2;

-- name: CreateTag :one
INSERT INTO tags (
  user_id,
  name
) VALUES (
  $1, $2
)
ON CONFLICT (user_id, name) DO UPDATE SET name = EXCLUDED.name
RETURNING *;

-- name: ListTags :many
SELECT * FROM tags
WHERE user_id = $1
ORDER BY name ASC;

-- name: AttachTagToTask :exec
INSERT INTO task_tags (task_id, tag_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: DetachTagFromTask :exec
DELETE FROM task_tags
WHERE task_id = $1 AND tag_id = $2;

-- name: ListTagsByTaskID :many
SELECT tg.* FROM tags tg
JOIN task_tags tt ON tg.id = tt.tag_id
WHERE tt.task_id = $1
ORDER BY tg.name ASC;

-- name: DeleteTaskTagsByTaskID :exec
DELETE FROM task_tags
WHERE task_id = $1;

-- name: CreateTimeBlock :one
INSERT INTO time_blocks (
  user_id,
  task_id,
  title,
  starts_at,
  ends_at
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetTimeBlockByID :one
SELECT * FROM time_blocks
WHERE id = $1 AND user_id = $2
LIMIT 1;

-- name: ListTimeBlocks :many
SELECT * FROM time_blocks
WHERE user_id = $1
  AND starts_at >= $2
  AND ends_at <= $3
ORDER BY starts_at ASC;

-- name: UpdateTimeBlock :one
UPDATE time_blocks
SET
  task_id = $3,
  title = $4,
  starts_at = $5,
  ends_at = $6
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteTimeBlock :exec
DELETE FROM time_blocks
WHERE id = $1 AND user_id = $2;

-- name: CheckTimeBlockOverlap :one
SELECT count(*) FROM time_blocks
WHERE user_id = $1
  AND id != $2
  AND starts_at < sqlc.arg('proposed_ends_at')
  AND ends_at > sqlc.arg('proposed_starts_at');

-- name: GetTasksForRange :many
SELECT id, title, due_at, estimate_minutes, status, priority FROM tasks
WHERE user_id = $1 AND due_at >= $2 AND due_at <= $3
ORDER BY due_at ASC;

-- name: GetOverdueTasks :many
SELECT * FROM tasks
WHERE user_id = $1 AND status != 'done' AND due_at < $2
ORDER BY due_at ASC;

-- name: GetTodayTasks :many
SELECT * FROM tasks
WHERE user_id = $1 AND due_at >= $2 AND due_at <= $3
ORDER BY priority ASC, due_at ASC;

-- name: GetUpcomingTasks :many
SELECT * FROM tasks
WHERE user_id = $1 AND due_at > $2 AND due_at <= $3
ORDER BY due_at ASC;
