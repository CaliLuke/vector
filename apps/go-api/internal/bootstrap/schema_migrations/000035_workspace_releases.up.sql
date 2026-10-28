BEGIN;
SET LOCAL search_path = public, pg_catalog;
CREATE TABLE release_tags (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 version varchar(255) NOT NULL CHECK(length(btrim(version))>0),description text NOT NULL DEFAULT '',commit_hash varchar(255) NOT NULL DEFAULT '',git_tag varchar(255) NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),deleted_at timestamptz,
 created_by_id uuid REFERENCES users(id) ON DELETE SET NULL,updated_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 UNIQUE(id,workspace_id)
);
CREATE UNIQUE INDEX release_tag_version_live ON release_tags(workspace_id,version) WHERE deleted_at IS NULL;
CREATE TABLE release_labels (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 name varchar(255) NOT NULL CHECK(length(btrim(name))>0),color varchar(32) NOT NULL DEFAULT '#4E5355',sort_order double precision NOT NULL DEFAULT 65535,
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),deleted_at timestamptz,
 created_by_id uuid REFERENCES users(id) ON DELETE SET NULL,updated_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 UNIQUE(id,workspace_id)
);
CREATE UNIQUE INDEX release_label_name_live ON release_labels(workspace_id,name) WHERE deleted_at IS NULL;
ALTER TABLE work_item_releases ALTER COLUMN project_id DROP NOT NULL;
ALTER TABLE work_item_releases ADD COLUMN description_id uuid NOT NULL DEFAULT gen_random_uuid(),
 ADD COLUMN description_html text NOT NULL DEFAULT '<p></p>',ADD COLUMN description_json jsonb NOT NULL DEFAULT '{}'::jsonb,
 ADD COLUMN status varchar(16) NOT NULL DEFAULT 'unreleased' CHECK(status IN('unreleased','released','cancelled')),
 ADD COLUMN release_date date,ADD COLUMN lead_id uuid REFERENCES users(id) ON DELETE SET NULL,ADD COLUMN tag_id uuid,
 ADD COLUMN is_prerelease boolean NOT NULL DEFAULT false,ADD COLUMN external_source varchar(255),ADD COLUMN external_id varchar(255),
 ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now(),ADD COLUMN deleted_at timestamptz,
 ADD COLUMN created_by_id uuid REFERENCES users(id) ON DELETE SET NULL,ADD COLUMN updated_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 ADD CONSTRAINT release_tag_owner FOREIGN KEY(tag_id,workspace_id) REFERENCES release_tags(id,workspace_id);
CREATE UNIQUE INDEX work_item_release_owner ON work_item_releases(id,workspace_id);
CREATE UNIQUE INDEX workspace_release_name_live ON work_item_releases(workspace_id,name) WHERE project_id IS NULL AND deleted_at IS NULL;
CREATE TABLE release_changelogs (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),release_id uuid NOT NULL UNIQUE,workspace_id uuid NOT NULL,
 description_html text NOT NULL DEFAULT '<p></p>',description_json jsonb NOT NULL DEFAULT '{}'::jsonb,description_stripped text NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),
 created_by_id uuid REFERENCES users(id) ON DELETE SET NULL,updated_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 FOREIGN KEY(release_id,workspace_id) REFERENCES work_item_releases(id,workspace_id) ON DELETE CASCADE
);
CREATE TABLE release_item_labels (
 release_id uuid NOT NULL,label_id uuid NOT NULL,workspace_id uuid NOT NULL,
 PRIMARY KEY(release_id,label_id),
 FOREIGN KEY(release_id,workspace_id) REFERENCES work_item_releases(id,workspace_id) ON DELETE CASCADE,
 FOREIGN KEY(label_id,workspace_id) REFERENCES release_labels(id,workspace_id) ON DELETE CASCADE
);
CREATE TABLE release_work_items (
 release_id uuid NOT NULL,issue_id uuid NOT NULL,project_id uuid NOT NULL,workspace_id uuid NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),created_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 PRIMARY KEY(release_id,issue_id),
 FOREIGN KEY(release_id,workspace_id) REFERENCES work_item_releases(id,workspace_id) ON DELETE CASCADE,
 FOREIGN KEY(issue_id,project_id,workspace_id) REFERENCES issues(id,project_id,workspace_id) ON UPDATE CASCADE ON DELETE CASCADE
);
CREATE INDEX release_work_items_issue ON release_work_items(issue_id);
CREATE TABLE release_events (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 actor_id uuid REFERENCES users(id) ON DELETE SET NULL,resource_id uuid NOT NULL,resource_kind text NOT NULL,action text NOT NULL,
 before_value jsonb,after_value jsonb,created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE release_revisions(workspace_id uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,revision bigint NOT NULL DEFAULT 1 CHECK(revision>0));
INSERT INTO release_revisions(workspace_id) SELECT id FROM workspaces;
CREATE FUNCTION seed_release_revision() RETURNS trigger LANGUAGE plpgsql SET search_path=public,pg_catalog AS $$
BEGIN INSERT INTO release_revisions(workspace_id) VALUES(NEW.id); RETURN NEW; END; $$;
CREATE TRIGGER workspace_release_revision AFTER INSERT ON workspaces FOR EACH ROW EXECUTE FUNCTION seed_release_revision();
CREATE FUNCTION advance_release_revision() RETURNS trigger LANGUAGE plpgsql SET search_path=public,pg_catalog AS $$
DECLARE old_workspace uuid;new_workspace uuid;
BEGIN
 IF TG_OP<>'INSERT' THEN old_workspace:=OLD.workspace_id; END IF;
 IF TG_OP<>'DELETE' THEN new_workspace:=NEW.workspace_id; END IF;
 PERFORM workspace_id FROM release_revisions WHERE workspace_id IN(old_workspace,new_workspace) ORDER BY workspace_id FOR UPDATE;
 UPDATE release_revisions SET revision=revision+1 WHERE workspace_id IN(old_workspace,new_workspace);
 RETURN NULL;
END; $$;
CREATE TRIGGER release_revision AFTER INSERT OR UPDATE OR DELETE ON work_item_releases FOR EACH ROW EXECUTE FUNCTION advance_release_revision();
CREATE TRIGGER release_tag_revision AFTER INSERT OR UPDATE OR DELETE ON release_tags FOR EACH ROW EXECUTE FUNCTION advance_release_revision();
CREATE TRIGGER release_label_revision AFTER INSERT OR UPDATE OR DELETE ON release_labels FOR EACH ROW EXECUTE FUNCTION advance_release_revision();
CREATE TRIGGER release_changelog_revision AFTER INSERT OR UPDATE OR DELETE ON release_changelogs FOR EACH ROW EXECUTE FUNCTION advance_release_revision();
CREATE TRIGGER release_item_label_revision AFTER INSERT OR UPDATE OR DELETE ON release_item_labels FOR EACH ROW EXECUTE FUNCTION advance_release_revision();
CREATE TRIGGER release_work_item_revision AFTER INSERT OR UPDATE OR DELETE ON release_work_items FOR EACH ROW EXECUTE FUNCTION advance_release_revision();
CREATE TRIGGER release_item_edit_revision AFTER INSERT OR UPDATE OR DELETE ON release_work_items FOR EACH ROW EXECUTE FUNCTION advance_issue_assignment_revision();
COMMIT;
