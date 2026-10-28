BEGIN;
SET LOCAL search_path = public, pg_catalog;
CREATE TABLE label_edit_revisions (
 label_id uuid PRIMARY KEY REFERENCES labels(id) ON DELETE CASCADE,
 revision bigint NOT NULL DEFAULT 1 CHECK(revision>0)
);
INSERT INTO label_edit_revisions(label_id) SELECT id FROM labels;
CREATE FUNCTION advance_label_edit_revision() RETURNS trigger
LANGUAGE plpgsql SET search_path=public,pg_catalog AS $$
BEGIN
 IF TG_OP='INSERT' THEN
  INSERT INTO label_edit_revisions(label_id) VALUES(NEW.id);
 ELSE
  UPDATE label_edit_revisions SET revision=revision+1 WHERE label_id=NEW.id;
  IF NOT FOUND THEN RAISE EXCEPTION 'label revision is missing'; END IF;
 END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER label_edit_revision AFTER INSERT OR UPDATE ON labels
 FOR EACH ROW EXECUTE FUNCTION advance_label_edit_revision();
COMMIT;
