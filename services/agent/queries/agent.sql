-- Conversations
-- name: CreateConversation :one
INSERT INTO conversations (user_id, title)
VALUES ($1, $2)
RETURNING *;

-- name: GetConversationByID :one
SELECT * FROM conversations
WHERE id = $1 AND user_id = $2;

-- name: ListConversationsByUserID :many
SELECT * FROM conversations
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: DeleteConversation :exec
DELETE FROM conversations
WHERE id = $1 AND user_id = $2;

-- Messages
-- name: CreateMessage :one
INSERT INTO messages (conversation_id, role, content)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListMessagesByConversationID :many
SELECT * FROM messages
WHERE conversation_id = $1
ORDER BY created_at ASC;

-- name: CountMessagesForUserSince :one
SELECT COUNT(*) FROM messages m
JOIN conversations c ON m.conversation_id = c.id
WHERE c.user_id = $1
  AND m.role = 'user'
  AND m.created_at >= $2;

-- Agent Runs
-- name: CreateAgentRun :one
INSERT INTO agent_runs (
  conversation_id, user_id, trigger, status, iterations, input_tokens, output_tokens, error
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateAgentRunStatus :one
UPDATE agent_runs
SET status = $2,
    iterations = $3,
    input_tokens = $4,
    output_tokens = $5,
    error = $6
WHERE id = $1
RETURNING *;

-- name: GetAgentRunByID :one
SELECT * FROM agent_runs
WHERE id = $1 AND user_id = $2;

-- Tool Calls
-- name: CreateToolCall :one
INSERT INTO tool_calls (run_id, tool_name, input, output, is_error)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListToolCallsByRunID :many
SELECT * FROM tool_calls
WHERE run_id = $1
ORDER BY created_at ASC;

-- Proposals
-- name: CreateProposal :one
INSERT INTO proposals (user_id, run_id, summary, actions, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetProposalByID :one
SELECT * FROM proposals
WHERE id = $1 AND user_id = $2;

-- name: ListProposalsByUserID :many
SELECT * FROM proposals
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateProposalStatus :one
UPDATE proposals
SET status = $2,
    actions = COALESCE($3, actions),
    decided_at = $4
WHERE id = $1 AND user_id = $5
RETURNING *;

-- Briefing Log
-- name: RecordBriefing :exec
INSERT INTO briefing_log (user_id, briefing_date)
VALUES ($1, $2)
ON CONFLICT (user_id, briefing_date) DO NOTHING;

-- name: HasBriefingForDate :one
SELECT EXISTS (
  SELECT 1 FROM briefing_log
  WHERE user_id = $1 AND briefing_date = $2
);
