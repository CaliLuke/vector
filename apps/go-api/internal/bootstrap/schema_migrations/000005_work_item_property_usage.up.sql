BEGIN;
SET LOCAL search_path = public, pg_catalog;
ALTER TABLE work_item_properties ADD COLUMN used boolean NOT NULL DEFAULT false;
UPDATE work_item_properties p SET used=true WHERE
 EXISTS(SELECT 1 FROM issues i WHERE i.workspace_id=p.workspace_id AND i.custom_properties ? p.id::text)
 OR EXISTS(SELECT 1 FROM draft_issues d WHERE d.workspace_id=p.workspace_id AND d.custom_properties ? p.id::text)
 OR EXISTS(SELECT 1 FROM work_item_property_versions v WHERE v.workspace_id=p.workspace_id AND v.property_id=p.id);

-- Usage survives clearing a value or deleting an item, including private drafts.
CREATE FUNCTION record_work_item_property_usage() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 PERFORM work_item_schema_lock(NEW.workspace_id);
 UPDATE work_item_properties SET used=true
 WHERE workspace_id=NEW.workspace_id AND NOT used AND NEW.custom_properties ? id::text
 AND NEW.custom_properties->id::text NOT IN ('null'::jsonb,'""'::jsonb,'[]'::jsonb);
 RETURN NEW;
END;
$$;
CREATE TRIGGER issue_property_usage BEFORE INSERT OR UPDATE OF custom_properties ON issues
 FOR EACH ROW EXECUTE FUNCTION record_work_item_property_usage();
CREATE TRIGGER draft_property_usage BEFORE INSERT OR UPDATE OF custom_properties ON draft_issues
 FOR EACH ROW EXECUTE FUNCTION record_work_item_property_usage();
COMMIT;
