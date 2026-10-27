BEGIN;
SET LOCAL search_path = public, pg_catalog;

CREATE UNIQUE INDEX projects_milestone_owner_key ON projects(id, workspace_id);
CREATE UNIQUE INDEX issues_milestone_owner_key ON issues(id, project_id, workspace_id);

CREATE TABLE milestones (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 project_id uuid NOT NULL,
 workspace_id uuid NOT NULL,
 title varchar(255) COLLATE "C" NOT NULL,
 target_date date,
 external_source varchar(255),
 external_id varchar(255),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 created_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 updated_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 deleted_at timestamptz,
 live_id uuid GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN id END) STORED UNIQUE,
 CONSTRAINT milestone_title_normalized CHECK (title <> '' AND title = btrim(title, E' \t\n\r\013\f')),
 CONSTRAINT milestone_calendar_date CHECK (target_date BETWEEN DATE '0001-01-01' AND DATE '9999-12-31'),
 CONSTRAINT milestone_project_owner FOREIGN KEY(project_id, workspace_id) REFERENCES projects(id, workspace_id) ON DELETE CASCADE,
 UNIQUE(id, project_id, workspace_id)
);
CREATE UNIQUE INDEX milestones_live_title ON milestones(project_id, title) WHERE deleted_at IS NULL;
CREATE INDEX milestones_live_page ON milestones(project_id, target_date, id) WHERE deleted_at IS NULL;

CREATE TABLE milestone_issues (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 milestone_id uuid NOT NULL,
 issue_id uuid NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
 project_id uuid NOT NULL,
 workspace_id uuid NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 created_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 updated_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 deleted_at timestamptz,
 -- Historical links do not prevent a later item move. Live links do.
 live_issue_id uuid GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN issue_id END) STORED,
 live_milestone_id uuid GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN milestone_id END) STORED,
 CONSTRAINT milestone_link_live_target FOREIGN KEY(live_milestone_id) REFERENCES milestones(live_id) ON DELETE CASCADE,
 CONSTRAINT milestone_link_owner FOREIGN KEY(milestone_id, project_id, workspace_id) REFERENCES milestones(id, project_id, workspace_id) ON DELETE CASCADE,
 CONSTRAINT milestone_item_owner FOREIGN KEY(live_issue_id, project_id, workspace_id) REFERENCES issues(id, project_id, workspace_id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX milestone_issues_live_item ON milestone_issues(issue_id) WHERE deleted_at IS NULL;
CREATE INDEX milestone_issues_live_page ON milestone_issues(milestone_id, created_at, id) WHERE deleted_at IS NULL;

CREATE TABLE milestone_edit_revisions (
 milestone_id uuid PRIMARY KEY REFERENCES milestones(id) ON DELETE CASCADE,
 revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0)
);

CREATE FUNCTION guard_milestone_row() RETURNS trigger
LANGUAGE plpgsql SET search_path = public, pg_catalog AS $$
BEGIN
 IF (NEW.id, NEW.project_id, NEW.workspace_id) IS DISTINCT FROM (OLD.id, OLD.project_id, OLD.workspace_id) THEN
  RAISE EXCEPTION 'milestone ownership is immutable' USING ERRCODE = '23514';
 END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER milestone_row_guard BEFORE UPDATE ON milestones
 FOR EACH ROW EXECUTE FUNCTION guard_milestone_row();

CREATE FUNCTION guard_milestone_link() RETURNS trigger
LANGUAGE plpgsql SET search_path = public, pg_catalog AS $$
BEGIN
 IF TG_OP = 'UPDATE' AND NEW.id IS DISTINCT FROM OLD.id THEN
  RAISE EXCEPTION 'milestone association identity is immutable' USING ERRCODE = '23514';
 END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER milestone_link_guard BEFORE INSERT OR UPDATE ON milestone_issues
 FOR EACH ROW EXECUTE FUNCTION guard_milestone_link();

CREATE FUNCTION advance_milestone_edit_revision() RETURNS trigger
LANGUAGE plpgsql SET search_path = public, pg_catalog AS $$
BEGIN
 IF TG_OP = 'INSERT' THEN
  INSERT INTO milestone_edit_revisions(milestone_id) VALUES(NEW.id);
 ELSIF NEW IS DISTINCT FROM OLD THEN
  UPDATE milestone_edit_revisions SET revision=revision+1 WHERE milestone_id=NEW.id;
  IF NOT FOUND THEN
   RAISE EXCEPTION 'milestone revision is missing';
  END IF;
 END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER milestone_edit_revision AFTER INSERT OR UPDATE ON milestones
 FOR EACH ROW EXECUTE FUNCTION advance_milestone_edit_revision();

CREATE FUNCTION advance_milestone_link_revisions() RETURNS trigger
LANGUAGE plpgsql SET search_path = public, pg_catalog AS $$
DECLARE
 previous_milestone uuid;
 next_milestone uuid;
 affected uuid;
BEGIN
 IF TG_OP <> 'INSERT' AND OLD.deleted_at IS NULL THEN previous_milestone := OLD.milestone_id; END IF;
 IF TG_OP <> 'DELETE' AND NEW.deleted_at IS NULL THEN next_milestone := NEW.milestone_id; END IF;
 -- Metadata-only updates do not change the live association set.
 IF TG_OP = 'UPDATE' AND previous_milestone IS NOT DISTINCT FROM next_milestone AND NEW.issue_id=OLD.issue_id THEN
  RETURN NULL;
 END IF;
 FOR affected IN SELECT DISTINCT id FROM unnest(ARRAY[previous_milestone, next_milestone]) AS ids(id) WHERE id IS NOT NULL ORDER BY id LOOP
  UPDATE milestone_edit_revisions SET revision=revision+1 WHERE milestone_id=affected;
  IF NOT FOUND AND EXISTS(SELECT 1 FROM milestones WHERE id=affected) THEN
   RAISE EXCEPTION 'milestone revision is missing';
  END IF;
 END LOOP;
 RETURN NULL;
END;
$$;
CREATE TRIGGER milestone_link_revisions AFTER INSERT OR UPDATE OR DELETE ON milestone_issues
 FOR EACH ROW EXECUTE FUNCTION advance_milestone_link_revisions();
COMMIT;
