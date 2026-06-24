ALTER TABLE subscriptions
    DROP CONSTRAINT IF EXISTS subscriptions_repository_id_fkey;

DROP TABLE IF EXISTS repositories;
