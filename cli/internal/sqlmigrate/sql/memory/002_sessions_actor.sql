ALTER TABLE sessions ADD COLUMN actor VARCHAR(128) NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS sessions_actor_open ON sessions(actor) WHERE ended_at IS NULL AND actor != '';
