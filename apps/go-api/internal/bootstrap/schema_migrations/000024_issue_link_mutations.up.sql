BEGIN;
SET LOCAL search_path = public, pg_catalog;
CREATE TABLE issue_link_edit_revisions(link_id uuid PRIMARY KEY REFERENCES issue_links(id) ON DELETE CASCADE,revision bigint NOT NULL DEFAULT 1 CHECK(revision>0));
INSERT INTO issue_link_edit_revisions(link_id) SELECT id FROM issue_links;
CREATE FUNCTION advance_issue_link_edit_revision() RETURNS trigger LANGUAGE plpgsql SET search_path=public,pg_catalog AS $$
BEGIN
 IF TG_OP='INSERT' THEN INSERT INTO issue_link_edit_revisions(link_id) VALUES(NEW.id);
 ELSE UPDATE issue_link_edit_revisions SET revision=revision+1 WHERE link_id=NEW.id;
 IF NOT FOUND THEN RAISE EXCEPTION 'issue link revision is missing'; END IF;
 END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER issue_link_edit_revision AFTER INSERT OR UPDATE ON issue_links FOR EACH ROW EXECUTE FUNCTION advance_issue_link_edit_revision();
COMMIT;
