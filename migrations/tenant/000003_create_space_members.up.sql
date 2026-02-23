CREATE TABLE space_members (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    space_id    UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id),
    role        TEXT NOT NULL DEFAULT 'editor',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(space_id, user_id)
);

CREATE INDEX idx_space_members_space ON space_members(space_id);
CREATE INDEX idx_space_members_user ON space_members(user_id);
