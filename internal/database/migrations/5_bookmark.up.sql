CREATE TABLE IF NOT EXISTS bookmarks
(
    form_id UUID REFERENCES forms (id),
    user_id UUID REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (form_id, user_id)
)