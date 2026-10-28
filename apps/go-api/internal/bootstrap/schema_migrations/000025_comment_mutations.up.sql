BEGIN;
SET LOCAL search_path = public, pg_catalog;
CREATE TABLE issue_comment_edit_revisions(comment_id uuid PRIMARY KEY REFERENCES issue_comments(id) ON DELETE CASCADE,revision bigint NOT NULL DEFAULT 1 CHECK(revision>0));
INSERT INTO issue_comment_edit_revisions(comment_id) SELECT id FROM issue_comments;
CREATE FUNCTION advance_issue_comment_edit_revision() RETURNS trigger LANGUAGE plpgsql SET search_path=public,pg_catalog AS $$
BEGIN
 IF TG_OP='INSERT' THEN INSERT INTO issue_comment_edit_revisions(comment_id) VALUES(NEW.id);
 ELSE UPDATE issue_comment_edit_revisions SET revision=revision+1 WHERE comment_id=NEW.id;
 IF NOT FOUND THEN RAISE EXCEPTION 'issue comment revision is missing'; END IF;
 END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER issue_comment_edit_revision AFTER INSERT OR UPDATE ON issue_comments FOR EACH ROW EXECUTE FUNCTION advance_issue_comment_edit_revision();
ALTER TABLE work_item_effects DROP CONSTRAINT work_item_effects_kind_check;
ALTER TABLE work_item_effects ADD CONSTRAINT work_item_effects_kind_check CHECK(kind IN ('activity','description_version','soft_delete','webhook','asset_storage_delete','asset_storage_metadata','link_preview'));
COMMIT;
