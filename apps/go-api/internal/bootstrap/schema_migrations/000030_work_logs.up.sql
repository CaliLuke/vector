BEGIN;
SET LOCAL search_path = public, pg_catalog;
CREATE TABLE work_logs (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 issue_id uuid NOT NULL,
 project_id uuid NOT NULL,
 workspace_id uuid NOT NULL,
 duration integer NOT NULL CHECK(duration > 0),
 description text NOT NULL DEFAULT '',
 logged_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 created_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 updated_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 deleted_at timestamptz,
 CONSTRAINT work_log_item_owner FOREIGN KEY(issue_id,project_id,workspace_id)
  REFERENCES issues(id,project_id,workspace_id) ON UPDATE CASCADE ON DELETE CASCADE
);
CREATE INDEX work_logs_live_item_page ON work_logs(issue_id,created_at DESC,id) WHERE deleted_at IS NULL;
CREATE TABLE work_log_edit_revisions (
 work_log_id uuid PRIMARY KEY REFERENCES work_logs(id) ON DELETE CASCADE,
 revision bigint NOT NULL DEFAULT 1 CHECK(revision > 0)
);
CREATE FUNCTION advance_work_log_revision() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 INSERT INTO work_log_edit_revisions(work_log_id) VALUES(NEW.id)
 ON CONFLICT(work_log_id) DO UPDATE SET revision=work_log_edit_revisions.revision+1;
 RETURN NEW;
END $$;
CREATE TRIGGER work_log_revision AFTER INSERT OR UPDATE ON work_logs FOR EACH ROW EXECUTE FUNCTION advance_work_log_revision();
COMMIT;
