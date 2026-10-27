BEGIN;
SET LOCAL search_path = public, pg_catalog;
CREATE TABLE project_edit_revisions (
 project_id uuid PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
 revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0)
);
INSERT INTO project_edit_revisions(project_id) SELECT id FROM projects;

-- Every project writer participates, including HTTP, workers and collaboration.
-- Keep the counter outside projects so existing row projections do not change.
CREATE FUNCTION advance_project_edit_revision() RETURNS trigger
LANGUAGE plpgsql SET search_path = public, pg_catalog AS $$
BEGIN
 IF TG_OP = 'INSERT' THEN
  INSERT INTO project_edit_revisions(project_id) VALUES(NEW.id);
 ELSE
  UPDATE project_edit_revisions SET revision=revision+1 WHERE project_id=NEW.id;
  IF NOT FOUND THEN
   RAISE EXCEPTION 'project revision is missing';
  END IF;
 END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER project_edit_revision AFTER INSERT OR UPDATE ON projects
 FOR EACH ROW EXECUTE FUNCTION advance_project_edit_revision();
COMMIT;
