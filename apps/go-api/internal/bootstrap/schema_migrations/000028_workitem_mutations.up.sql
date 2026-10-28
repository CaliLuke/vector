BEGIN;
SET LOCAL search_path = public, pg_catalog;
CREATE TABLE issue_edit_revisions(issue_id uuid PRIMARY KEY REFERENCES issues(id) ON DELETE CASCADE,revision bigint NOT NULL DEFAULT 1 CHECK(revision>0));
INSERT INTO issue_edit_revisions(issue_id) SELECT id FROM issues;
CREATE FUNCTION advance_issue_edit_revision() RETURNS trigger LANGUAGE plpgsql SET search_path=public,pg_catalog AS $$
BEGIN
 IF TG_OP='INSERT' THEN INSERT INTO issue_edit_revisions(issue_id) VALUES(NEW.id);
 ELSE UPDATE issue_edit_revisions SET revision=revision+1 WHERE issue_id=NEW.id;
  IF NOT FOUND THEN RAISE EXCEPTION 'issue revision is missing'; END IF;
 END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER issue_edit_revision AFTER INSERT OR UPDATE ON issues FOR EACH ROW EXECUTE FUNCTION advance_issue_edit_revision();
CREATE FUNCTION advance_issue_assignment_revision() RETURNS trigger LANGUAGE plpgsql SET search_path=public,pg_catalog AS $$
DECLARE old_issue uuid; new_issue uuid;
BEGIN
 IF TG_OP<>'INSERT' THEN old_issue:=OLD.issue_id; END IF;
 IF TG_OP<>'DELETE' THEN new_issue:=NEW.issue_id; END IF;
 PERFORM issue_id FROM issue_edit_revisions WHERE issue_id IN(old_issue,new_issue) ORDER BY issue_id FOR UPDATE;
 UPDATE issue_edit_revisions SET revision=revision+1 WHERE issue_id IN(old_issue,new_issue);
 RETURN NULL;
END;
$$;
CREATE TRIGGER issue_assignee_edit_revision AFTER INSERT OR UPDATE OR DELETE ON issue_assignees FOR EACH ROW EXECUTE FUNCTION advance_issue_assignment_revision();
CREATE TRIGGER issue_label_edit_revision AFTER INSERT OR UPDATE OR DELETE ON issue_labels FOR EACH ROW EXECUTE FUNCTION advance_issue_assignment_revision();
COMMIT;
