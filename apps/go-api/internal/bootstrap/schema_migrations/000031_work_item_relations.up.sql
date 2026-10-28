BEGIN;
SET LOCAL search_path = public, pg_catalog;
DROP INDEX issue_relation_unique_issue_related_issue_when_deleted_at_null;
CREATE UNIQUE INDEX issue_relation_unique_kind_live ON issue_relations(issue_id,related_issue_id,relation_type) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX issues_relation_owner_key ON issues(id,workspace_id);
CREATE TABLE work_item_relation_definitions (
 id uuid PRIMARY KEY, workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 name varchar(255) NOT NULL CHECK(length(btrim(name))>0),
 outward varchar(255) NOT NULL CHECK(length(btrim(outward))>0),
 inward varchar(255) NOT NULL CHECK(length(btrim(inward))>0),
 is_default boolean NOT NULL DEFAULT false, is_active boolean NOT NULL DEFAULT true,
 color varchar(255) NOT NULL DEFAULT '', sort_order double precision NOT NULL DEFAULT 65535,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), deleted_at timestamptz,
 created_by_id uuid REFERENCES users(id) ON DELETE SET NULL, updated_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 UNIQUE(id,workspace_id)
);
CREATE UNIQUE INDEX work_item_relation_definition_name ON work_item_relation_definitions(workspace_id,lower(name)) WHERE deleted_at IS NULL;
CREATE TABLE work_item_custom_relations (
 id uuid PRIMARY KEY, workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 definition_id uuid NOT NULL, issue_id uuid NOT NULL, related_issue_id uuid NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), deleted_at timestamptz,
 created_by_id uuid REFERENCES users(id) ON DELETE SET NULL, updated_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 FOREIGN KEY(definition_id,workspace_id) REFERENCES work_item_relation_definitions(id,workspace_id) ON DELETE CASCADE,
 FOREIGN KEY(issue_id,workspace_id) REFERENCES issues(id,workspace_id) ON DELETE CASCADE,
 FOREIGN KEY(related_issue_id,workspace_id) REFERENCES issues(id,workspace_id) ON DELETE CASCADE,
 CHECK(issue_id<>related_issue_id)
);
CREATE UNIQUE INDEX work_item_custom_relation_live ON work_item_custom_relations(definition_id,issue_id,related_issue_id) WHERE deleted_at IS NULL;
CREATE INDEX work_item_custom_relation_target ON work_item_custom_relations(related_issue_id) WHERE deleted_at IS NULL;
CREATE TABLE work_item_relation_revisions(workspace_id uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE, revision bigint NOT NULL DEFAULT 1 CHECK(revision>0));
INSERT INTO work_item_relation_revisions(workspace_id) SELECT id FROM workspaces;
CREATE FUNCTION seed_work_item_relation_revision() RETURNS trigger LANGUAGE plpgsql SET search_path=public,pg_catalog AS $$
BEGIN INSERT INTO work_item_relation_revisions(workspace_id) VALUES(NEW.id); RETURN NEW; END; $$;
CREATE TRIGGER workspace_relation_revision AFTER INSERT ON workspaces FOR EACH ROW EXECUTE FUNCTION seed_work_item_relation_revision();
CREATE FUNCTION advance_work_item_relation_revision() RETURNS trigger LANGUAGE plpgsql SET search_path=public,pg_catalog AS $$
DECLARE old_workspace uuid; new_workspace uuid;
BEGIN
 IF TG_OP<>'INSERT' THEN old_workspace:=OLD.workspace_id; END IF;
 IF TG_OP<>'DELETE' THEN new_workspace:=NEW.workspace_id; END IF;
 PERFORM workspace_id FROM work_item_relation_revisions WHERE workspace_id IN(old_workspace,new_workspace) ORDER BY workspace_id FOR UPDATE;
 UPDATE work_item_relation_revisions SET revision=revision+1 WHERE workspace_id IN(old_workspace,new_workspace);
 RETURN NULL;
END; $$;
CREATE TRIGGER dependency_relation_revision AFTER INSERT OR UPDATE OR DELETE ON issue_relations FOR EACH ROW EXECUTE FUNCTION advance_work_item_relation_revision();
CREATE TRIGGER custom_relation_revision AFTER INSERT OR UPDATE OR DELETE ON work_item_custom_relations FOR EACH ROW EXECUTE FUNCTION advance_work_item_relation_revision();
CREATE TRIGGER relation_definition_revision AFTER INSERT OR UPDATE OR DELETE ON work_item_relation_definitions FOR EACH ROW EXECUTE FUNCTION advance_work_item_relation_revision();
COMMIT;
