-- name: CreateReminder :one
INSERT INTO reminders (
  user_id,
  task_id,
  message,
  remind_at
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: ListRemindersByUserID :many
SELECT * FROM reminders
WHERE user_id = $1
ORDER BY remind_at ASC;

-- name: GetDueRemindersForUpdate :many
SELECT id, user_id, task_id, message, remind_at
FROM reminders
WHERE remind_at <= $1 AND fired = false
ORDER BY remind_at ASC
LIMIT $2
FOR UPDATE SKIP LOCKED;

-- name: MarkReminderFired :exec
UPDATE reminders
SET fired = true
WHERE id = $1;

-- name: CreateNotification :one
INSERT INTO notifications (
  user_id,
  kind,
  title,
  body
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: ListNotifications :many
SELECT * FROM notifications
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountNotifications :one
SELECT count(*) FROM notifications
WHERE user_id = $1;

-- name: MarkNotificationAsRead :one
UPDATE notifications
SET read = true
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: GetUnreadNotificationCount :one
SELECT count(*) FROM notifications
WHERE user_id = $1 AND read = false;
