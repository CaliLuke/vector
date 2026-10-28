ALTER TABLE public.pages ADD COLUMN collaboration_revision bigint NOT NULL DEFAULT 0 CHECK (collaboration_revision >= 0);
