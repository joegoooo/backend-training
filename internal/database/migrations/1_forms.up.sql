CREATE TABLE IF NOT EXISTS forms (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
title TEXT NOT NULL,
description TEXT,
created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);