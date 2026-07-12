\set ON_ERROR_STOP on

WITH inserted_subscribers AS (
    INSERT INTO subscribers (email)
    SELECT 'bench-user-' || n || '@example.com'
    FROM generate_series(1, :subscription_count::int) AS n
    ON CONFLICT (email) DO NOTHING
    RETURNING id
),
existing_subscribers AS (
    SELECT id
    FROM subscribers
    WHERE email LIKE 'bench-user-%@example.com'
),
bench_subscribers AS (
    SELECT id
    FROM inserted_subscribers
    UNION
    SELECT id
    FROM existing_subscribers
    ORDER BY id
    LIMIT :subscription_count::int
)
INSERT INTO subscriptions (subscriber_id, repository_id, subscription_status, token)
SELECT
    id,
    :repository_id::int,
    1,
    'bench-token-' || :repository_id::text || '-' || id::text
FROM bench_subscribers
ON CONFLICT (subscriber_id, repository_id) DO UPDATE
SET
    subscription_status = EXCLUDED.subscription_status,
    token = EXCLUDED.token;

SELECT
    :repository_id::int AS repository_id,
    count(*) AS active_subscription_count
FROM subscriptions
WHERE repository_id = :repository_id::int
  AND subscription_status = 1;
