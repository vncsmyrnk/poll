CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_unique_user_email ON users(email);

ALTER TABLE votes ADD COLUMN user_id UUID NOT NULL REFERENCES users(id);

DROP INDEX idx_unique_vote_per_ip;

CREATE UNIQUE INDEX idx_unique_vote_per_user ON votes(poll_id, user_id);
