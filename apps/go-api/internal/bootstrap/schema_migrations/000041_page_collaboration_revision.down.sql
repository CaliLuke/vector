DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM public.pages WHERE collaboration_revision > 0) THEN
    RAISE EXCEPTION 'cannot discard page collaboration revisions after external replacement';
  END IF;
END; $$;
ALTER TABLE public.pages DROP COLUMN collaboration_revision;
