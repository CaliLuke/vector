BEGIN;
SET LOCAL search_path = public, pg_catalog;
CREATE UNIQUE INDEX pages_owner_key ON pages(id,workspace_id);
CREATE TABLE page_collections (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 name varchar(255) NOT NULL CHECK(length(btrim(name))>0),owned_by_id uuid NOT NULL REFERENCES users(id),access smallint NOT NULL DEFAULT 0 CHECK(access IN(0,1)),
 logo_props jsonb NOT NULL DEFAULT '{}',sort_order double precision NOT NULL DEFAULT 65535,is_default boolean NOT NULL DEFAULT false,is_global boolean NOT NULL DEFAULT true,
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),deleted_at timestamptz,
 created_by_id uuid REFERENCES users(id) ON DELETE SET NULL,updated_by_id uuid REFERENCES users(id) ON DELETE SET NULL,UNIQUE(id,workspace_id)
);
CREATE TABLE page_collection_members (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),workspace_id uuid NOT NULL,collection_id uuid NOT NULL,member_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 access smallint NOT NULL CHECK(access IN(0,1,2)),created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),deleted_at timestamptz,
 created_by_id uuid REFERENCES users(id) ON DELETE SET NULL,updated_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 FOREIGN KEY(collection_id,workspace_id) REFERENCES page_collections(id,workspace_id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX page_collection_member_live ON page_collection_members(collection_id,member_id) WHERE deleted_at IS NULL;
CREATE TABLE page_collection_pages (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),workspace_id uuid NOT NULL,collection_id uuid NOT NULL,page_id uuid NOT NULL,sort_order double precision NOT NULL DEFAULT 65535,
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),created_by_id uuid REFERENCES users(id) ON DELETE SET NULL,updated_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 FOREIGN KEY(collection_id,workspace_id) REFERENCES page_collections(id,workspace_id) ON DELETE CASCADE,
 FOREIGN KEY(page_id,workspace_id) REFERENCES pages(id,workspace_id) ON DELETE CASCADE,UNIQUE(page_id)
);
CREATE TABLE work_item_pages (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),workspace_id uuid NOT NULL,project_id uuid NOT NULL,issue_id uuid NOT NULL,page_id uuid NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),deleted_at timestamptz,
 created_by_id uuid REFERENCES users(id) ON DELETE SET NULL,updated_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 FOREIGN KEY(issue_id,project_id,workspace_id) REFERENCES issues(id,project_id,workspace_id) ON UPDATE CASCADE ON DELETE CASCADE,
 FOREIGN KEY(page_id,workspace_id) REFERENCES pages(id,workspace_id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX work_item_page_live ON work_item_pages(issue_id,page_id) WHERE deleted_at IS NULL;
CREATE TABLE page_api_events (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,actor_id uuid REFERENCES users(id) ON DELETE SET NULL,
 resource_id uuid NOT NULL,resource_kind text NOT NULL,action text NOT NULL,before_value jsonb,after_value jsonb,created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE page_api_revisions(workspace_id uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,revision bigint NOT NULL DEFAULT 1 CHECK(revision>0));
INSERT INTO page_api_revisions(workspace_id) SELECT id FROM workspaces;
CREATE FUNCTION seed_page_api_revision() RETURNS trigger LANGUAGE plpgsql SET search_path=public,pg_catalog AS $$
BEGIN INSERT INTO page_api_revisions(workspace_id) VALUES(NEW.id);RETURN NEW;END; $$;
CREATE TRIGGER workspace_page_api_revision AFTER INSERT ON workspaces FOR EACH ROW EXECUTE FUNCTION seed_page_api_revision();
CREATE FUNCTION advance_page_api_revision() RETURNS trigger LANGUAGE plpgsql SET search_path=public,pg_catalog AS $$
DECLARE old_workspace uuid;new_workspace uuid;
BEGIN
 IF TG_OP<>'INSERT' THEN old_workspace:=OLD.workspace_id;END IF;
 IF TG_OP<>'DELETE' THEN new_workspace:=NEW.workspace_id;END IF;
 PERFORM workspace_id FROM page_api_revisions WHERE workspace_id IN(old_workspace,new_workspace) ORDER BY workspace_id FOR UPDATE;
 UPDATE page_api_revisions SET revision=revision+1 WHERE workspace_id IN(old_workspace,new_workspace);
 RETURN NULL;
END; $$;
CREATE TRIGGER page_api_revision AFTER INSERT OR UPDATE OR DELETE ON pages FOR EACH ROW EXECUTE FUNCTION advance_page_api_revision();
CREATE TRIGGER page_project_api_revision AFTER INSERT OR UPDATE OR DELETE ON project_pages FOR EACH ROW EXECUTE FUNCTION advance_page_api_revision();
CREATE TRIGGER collection_api_revision AFTER INSERT OR UPDATE OR DELETE ON page_collections FOR EACH ROW EXECUTE FUNCTION advance_page_api_revision();
CREATE TRIGGER collection_member_api_revision AFTER INSERT OR UPDATE OR DELETE ON page_collection_members FOR EACH ROW EXECUTE FUNCTION advance_page_api_revision();
CREATE TRIGGER collection_page_api_revision AFTER INSERT OR UPDATE OR DELETE ON page_collection_pages FOR EACH ROW EXECUTE FUNCTION advance_page_api_revision();
CREATE TRIGGER item_page_api_revision AFTER INSERT OR UPDATE OR DELETE ON work_item_pages FOR EACH ROW EXECUTE FUNCTION advance_page_api_revision();
CREATE TRIGGER item_page_edit_revision AFTER INSERT OR UPDATE OR DELETE ON work_item_pages FOR EACH ROW EXECUTE FUNCTION advance_issue_assignment_revision();
CREATE FUNCTION collection_access(collection uuid,actor uuid,workspace uuid) RETURNS smallint LANGUAGE sql STABLE SET search_path=public,pg_catalog AS $$
 SELECT CASE WHEN c.owned_by_id=actor THEN 2 ELSE COALESCE((SELECT m.access FROM page_collection_members m WHERE m.collection_id=c.id AND m.workspace_id=workspace AND m.member_id=actor AND m.deleted_at IS NULL),CASE WHEN c.access=0 THEN CASE WHEN wm.role IN(15,20) THEN 2 ELSE 0 END END) END::smallint
 FROM page_collections c JOIN workspace_members wm ON wm.workspace_id=c.workspace_id AND wm.member_id=actor JOIN users u ON u.id=actor AND u.is_active JOIN workspaces w ON w.id=c.workspace_id AND w.deleted_at IS NULL
 WHERE c.id=collection AND c.workspace_id=workspace AND c.deleted_at IS NULL AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20);
$$;
CREATE FUNCTION workspace_page_visible(page uuid,actor uuid,workspace uuid) RETURNS boolean LANGUAGE sql STABLE SET search_path=public,pg_catalog AS $$
 WITH RECURSIVE ancestors AS (
 SELECT p.id,p.parent_id,p.owned_by_id,p.access,p.is_global,p.deleted_at FROM pages p WHERE p.id=page AND p.workspace_id=workspace
 UNION SELECT p.id,p.parent_id,p.owned_by_id,p.access,p.is_global,p.deleted_at FROM pages p JOIN ancestors a ON a.parent_id=p.id WHERE p.workspace_id=workspace
 )
 SELECT EXISTS(SELECT 1 FROM ancestors) AND EXISTS(SELECT 1 FROM workspace_members wm JOIN users u ON u.id=wm.member_id AND u.is_active JOIN workspaces w ON w.id=wm.workspace_id AND w.deleted_at IS NULL WHERE wm.member_id=actor AND wm.workspace_id=workspace AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20))
 AND NOT EXISTS(SELECT 1 FROM ancestors p WHERE p.deleted_at IS NOT NULL OR NOT p.is_global OR NOT(p.owned_by_id=actor OR p.access=0)
 OR EXISTS(SELECT 1 FROM project_pages pp WHERE pp.page_id=p.id AND pp.deleted_at IS NULL)
 OR p.parent_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM ancestors a WHERE a.id=p.parent_id)
 OR EXISTS(SELECT 1 FROM page_collection_pages link WHERE link.page_id=p.id AND (link.workspace_id<>workspace OR collection_access(link.collection_id,actor,workspace) IS NULL)));
$$;
CREATE FUNCTION collection_access_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN IF OLD.access<>NEW.access THEN RAISE EXCEPTION 'collection access is immutable';END IF;RETURN NEW;END; $$;
CREATE TRIGGER collection_access_immutable BEFORE UPDATE ON page_collections FOR EACH ROW EXECUTE FUNCTION collection_access_immutable();
ALTER TABLE work_item_effects DROP CONSTRAINT work_item_effects_kind_check;
ALTER TABLE work_item_effects ADD CONSTRAINT work_item_effects_kind_check CHECK(kind IN('activity','description_version','soft_delete','webhook','asset_storage_delete','asset_storage_metadata','link_preview','asset_storage_upload','page_transaction','page_version'));
COMMIT;
