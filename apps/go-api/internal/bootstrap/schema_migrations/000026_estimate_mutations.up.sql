BEGIN;
SET LOCAL search_path = public, pg_catalog;
ALTER TABLE estimates ADD COLUMN external_source varchar(255), ADD COLUMN external_id varchar(255);
ALTER TABLE estimate_points ADD COLUMN external_source varchar(255), ADD COLUMN external_id varchar(255);
CREATE TABLE project_estimate_revisions (
 project_id uuid PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
 revision bigint NOT NULL DEFAULT 1 CHECK(revision>0)
);
INSERT INTO project_estimate_revisions(project_id) SELECT id FROM projects;
CREATE FUNCTION advance_project_estimate_revision() RETURNS trigger LANGUAGE plpgsql SET search_path=public,pg_catalog AS $$
BEGIN
 IF TG_OP='INSERT' THEN INSERT INTO project_estimate_revisions(project_id) VALUES(NEW.id);
 ELSIF NEW.estimate_id IS DISTINCT FROM OLD.estimate_id THEN
  UPDATE project_estimate_revisions SET revision=revision+1 WHERE project_id=NEW.id;
 END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER project_estimate_revision AFTER INSERT OR UPDATE OF estimate_id ON projects FOR EACH ROW EXECUTE FUNCTION advance_project_estimate_revision();
CREATE FUNCTION advance_estimate_catalog_revision() RETURNS trigger LANGUAGE plpgsql SET search_path=public,pg_catalog AS $$
DECLARE old_project uuid; new_project uuid;
BEGIN
 IF TG_OP<>'INSERT' THEN old_project:=OLD.project_id; END IF;
 IF TG_OP<>'DELETE' THEN new_project:=NEW.project_id; END IF;
 PERFORM project_id FROM project_estimate_revisions WHERE project_id IN(old_project,new_project) ORDER BY project_id FOR UPDATE;
 UPDATE project_estimate_revisions SET revision=revision+1 WHERE project_id IN(old_project,new_project);
 RETURN NULL;
END;
$$;
CREATE TRIGGER estimate_catalog_revision AFTER INSERT OR UPDATE OR DELETE ON estimates FOR EACH ROW EXECUTE FUNCTION advance_estimate_catalog_revision();
CREATE TRIGGER estimate_point_catalog_revision AFTER INSERT OR UPDATE OR DELETE ON estimate_points FOR EACH ROW EXECUTE FUNCTION advance_estimate_catalog_revision();
COMMIT;
