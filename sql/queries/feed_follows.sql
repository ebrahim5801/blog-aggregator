-- name: CreateFeedFollow :one
WITH inserted_feed_follow as (
  INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
  VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *
)
SELECT
    inserted_feed_follow.*,
    feeds.name AS feed_name,
    users.name AS user_name
FROM inserted_feed_follow
INNER JOIN users on users.id = inserted_feed_follow.user_id
INNER JOIN feeds on feeds.id = inserted_feed_follow.feed_id
;


-- name: GetFeedFollowsForUser :many
SELECT feed_follows.*, users.name as user_name, feeds.name as feed_name 
FROM feed_follows 
INNER JOIN users on users.id = feed_follows.user_id
INNER JOIN feeds on feeds.id = feed_follows.feed_id
where feed_follows.user_id = $1
;

-- name: DeleteFeedFollow :exec
DELETE FROM feed_follows
WHERE feed_id = $1 AND user_id = $2;

