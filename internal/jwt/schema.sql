CREATE TABLE IF NOT EXISTS jwt (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL references users(id),
    expiration_time TIMESTAMPTZ NOT NULL DEFAULT now(),
    is_available bool NOT NULL DEFAULT true
    );