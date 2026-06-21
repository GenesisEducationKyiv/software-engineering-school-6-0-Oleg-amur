ALTER TABLE subscriptions
    ADD COLUMN IF NOT EXISTS repository_name TEXT;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'subscriptions' AND column_name = 'repository_id'
    ) THEN
        EXECUTE '
            UPDATE subscriptions AS s
            SET repository_name = r.name
            FROM repositories AS r
            WHERE s.repository_id = r.id
              AND s.repository_name IS NULL';

        ALTER TABLE subscriptions
            DROP CONSTRAINT IF EXISTS subscriptions_subscriber_id_repository_id_key;

        ALTER TABLE subscriptions
            DROP CONSTRAINT IF EXISTS subscriptions_repository_id_fkey;

        ALTER TABLE subscriptions
            DROP COLUMN repository_id;
    END IF;
END $$;

ALTER TABLE subscriptions
    ALTER COLUMN repository_name SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS subscriptions_subscriber_repository_name_key
    ON subscriptions (subscriber_id, repository_name);

DROP TABLE IF EXISTS repositories;
