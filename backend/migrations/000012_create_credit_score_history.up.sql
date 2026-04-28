-- Credit score history table for tracking score trajectory over time.
-- Every score computation is logged here (in addition to the upsert in credit_scores).
-- Enables trend analysis, regulatory audits, and ML training data.

CREATE TABLE credit_score_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    crew_member_id  UUID NOT NULL,
    score           INT NOT NULL,
    grade           VARCHAR(20) NOT NULL,
    model_version   VARCHAR(50) NOT NULL,
    factors         JSONB,
    suggestions     JSONB,
    computed_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_csh_crew_date ON credit_score_history (crew_member_id, computed_at DESC);
CREATE INDEX idx_csh_score ON credit_score_history (score);
