BEGIN;
SET LOCAL search_path = public, pg_catalog;

CREATE TABLE state_edit_revisions (
 state_id uuid PRIMARY KEY REFERENCES states(id) ON DELETE CASCADE,
 revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0)
);
INSERT INTO state_edit_revisions(state_id) SELECT id FROM states;

CREATE FUNCTION advance_state_edit_revision() RETURNS trigger
LANGUAGE plpgsql SET search_path = public, pg_catalog AS $$
BEGIN
 IF TG_OP = 'INSERT' THEN
  INSERT INTO state_edit_revisions(state_id) VALUES(NEW.id);
 ELSE
  UPDATE state_edit_revisions SET revision=revision+1 WHERE state_id=NEW.id;
  IF NOT FOUND THEN
   RAISE EXCEPTION 'state revision is missing';
  END IF;
 END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER state_edit_revision AFTER INSERT OR UPDATE ON states
 FOR EACH ROW EXECUTE FUNCTION advance_state_edit_revision();

-- Check the final row, not an earlier queued trigger image: an item can change
-- state, be soft-deleted, or be removed later in the same transaction. Deferral
-- also retains the existing FK contract for parent creation order.
CREATE FUNCTION require_live_item_state() RETURNS trigger
LANGUAGE plpgsql SET search_path = public, pg_catalog AS $$
DECLARE selected_state uuid;
BEGIN
 EXECUTE format('SELECT state_id FROM public.%I WHERE id=$1 AND deleted_at IS NULL', TG_TABLE_NAME)
  INTO selected_state USING NEW.id;
 IF selected_state IS NULL THEN
  RETURN NULL;
 END IF;
 -- State deletion takes FOR UPDATE before its subsequent occupancy query.
 PERFORM id FROM states WHERE id=selected_state AND deleted_at IS NULL FOR SHARE;
 IF NOT FOUND THEN
  RAISE EXCEPTION 'a live item must reference a live state'
   USING ERRCODE='23514', CONSTRAINT='item_state_must_be_live';
 END IF;
 RETURN NULL;
END;
$$;
CREATE CONSTRAINT TRIGGER issue_live_state
 AFTER INSERT OR UPDATE OF state_id,deleted_at ON issues
 DEFERRABLE INITIALLY DEFERRED FOR EACH ROW
 EXECUTE FUNCTION require_live_item_state();
CREATE CONSTRAINT TRIGGER draft_live_state
 AFTER INSERT OR UPDATE OF state_id,deleted_at ON draft_issues
 DEFERRABLE INITIALLY DEFERRED FOR EACH ROW
 EXECUTE FUNCTION require_live_item_state();
COMMIT;
