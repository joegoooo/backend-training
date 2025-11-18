CREATE TABLE IF NOT EXISTS forms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    description TEXT,
    author_id TEXT references users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
    );