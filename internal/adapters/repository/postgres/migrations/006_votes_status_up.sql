CREATE TYPE vote_status AS ENUM (
  'valid',
  'pending',
  'invalid'
);

ALTER TABLE votes ADD COLUMN status vote_status NOT NULL DEFAULT 'pending';

DROP INDEX idx_unique_vote_per_user;

CREATE UNIQUE INDEX idx_unique_vote_per_user_and_status ON votes(poll_id, user_id) WHERE deleted_at IS NULL;
