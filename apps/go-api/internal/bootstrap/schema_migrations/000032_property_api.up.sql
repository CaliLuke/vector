BEGIN;
SET LOCAL search_path = public, pg_catalog;
ALTER TABLE work_item_properties ADD COLUMN api_metadata jsonb NOT NULL DEFAULT '{}'::jsonb CHECK(jsonb_typeof(api_metadata)='object');
CREATE FUNCTION advance_property_api_revision() RETURNS trigger LANGUAGE plpgsql SET search_path=public,pg_catalog AS $$
DECLARE old_workspace uuid; new_workspace uuid;
BEGIN
 IF TG_TABLE_NAME='work_item_properties' THEN
  IF TG_OP<>'INSERT' THEN old_workspace:=OLD.workspace_id; END IF;
  IF TG_OP<>'DELETE' THEN new_workspace:=NEW.workspace_id; END IF;
 ELSE
  IF TG_OP<>'INSERT' THEN SELECT workspace_id INTO old_workspace FROM issue_types WHERE id=OLD.type_id; END IF;
  IF TG_OP<>'DELETE' THEN SELECT workspace_id INTO new_workspace FROM issue_types WHERE id=NEW.type_id; END IF;
 END IF;
 PERFORM workspace_id FROM work_item_type_api_revisions WHERE workspace_id IN(old_workspace,new_workspace) ORDER BY workspace_id FOR UPDATE;
 UPDATE work_item_type_api_revisions SET revision=revision+1 WHERE workspace_id IN(old_workspace,new_workspace);
 RETURN NULL;
END; $$;
CREATE TRIGGER property_api_revision AFTER INSERT OR UPDATE OR DELETE ON work_item_properties FOR EACH ROW EXECUTE FUNCTION advance_property_api_revision();
CREATE TRIGGER property_link_api_revision AFTER INSERT OR UPDATE OR DELETE ON work_item_type_properties FOR EACH ROW EXECUTE FUNCTION advance_property_api_revision();
COMMIT;
