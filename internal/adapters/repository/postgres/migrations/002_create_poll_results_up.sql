CREATE TABLE poll_results (
    poll_id UUID NOT NULL REFERENCES polls(id),
    option_id UUID NOT NULL REFERENCES poll_options(id),
    vote_count BIGINT NOT NULL DEFAULT 0,
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (poll_id, option_id)
);
