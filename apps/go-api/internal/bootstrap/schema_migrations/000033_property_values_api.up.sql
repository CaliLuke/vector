BEGIN;
SET LOCAL search_path = public, pg_catalog;
CREATE UNIQUE INDEX work_item_properties_api_owner ON work_item_properties(id,workspace_id);
CREATE TABLE work_item_property_value_identities(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 issue_id uuid NOT NULL,property_id uuid NOT NULL,project_id uuid NOT NULL,workspace_id uuid NOT NULL,
 value_key text NOT NULL DEFAULT '' CHECK(length(value_key)<=255),value_digest text NOT NULL,
 external_source varchar(255),external_id varchar(255),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),deleted_at timestamptz,
 FOREIGN KEY(issue_id,project_id,workspace_id) REFERENCES issues(id,project_id,workspace_id) ON UPDATE CASCADE ON DELETE CASCADE,
 FOREIGN KEY(property_id,workspace_id) REFERENCES work_item_properties(id,workspace_id) ON DELETE CASCADE,
 UNIQUE(issue_id,property_id,value_key)
);
-- Canonical values remain in issues.custom_properties. Digests distinguish
-- changed values without retaining a second copy of content in the API metadata.
CREATE FUNCTION sync_property_value_identities(item public.issues) RETURNS void LANGUAGE plpgsql SET search_path=public,pg_catalog AS $$
BEGIN
 WITH desired AS (
  SELECT p.id AS property_id,CASE WHEN jsonb_typeof(v.value)='array' THEN e.value#>>'{}' ELSE '' END AS value_key
  FROM jsonb_each(item.custom_properties) v JOIN work_item_properties p ON p.id::text=v.key AND p.workspace_id=item.workspace_id
  CROSS JOIN LATERAL jsonb_array_elements(CASE WHEN jsonb_typeof(v.value)='array' THEN v.value ELSE jsonb_build_array(v.value) END) e
  WHERE item.deleted_at IS NULL
 )
 UPDATE work_item_property_value_identities m SET deleted_at=clock_timestamp(),updated_at=clock_timestamp()
 WHERE m.issue_id=item.id AND m.deleted_at IS NULL AND NOT EXISTS(SELECT 1 FROM desired d WHERE d.property_id=m.property_id AND d.value_key=m.value_key);
 INSERT INTO work_item_property_value_identities(issue_id,property_id,project_id,workspace_id,value_key,value_digest)
 SELECT item.id,p.id,item.project_id,item.workspace_id,
  CASE WHEN jsonb_typeof(v.value)='array' THEN e.value#>>'{}' ELSE '' END,
  encode(sha256(convert_to(e.value::text,'UTF8')),'hex')
 FROM jsonb_each(item.custom_properties) v JOIN work_item_properties p ON p.id::text=v.key AND p.workspace_id=item.workspace_id
 CROSS JOIN LATERAL jsonb_array_elements(CASE WHEN jsonb_typeof(v.value)='array' THEN v.value ELSE jsonb_build_array(v.value) END) e
 WHERE item.deleted_at IS NULL
 ON CONFLICT(issue_id,property_id,value_key) DO UPDATE SET value_digest=excluded.value_digest,project_id=excluded.project_id,workspace_id=excluded.workspace_id,deleted_at=NULL,updated_at=clock_timestamp()
 WHERE work_item_property_value_identities.value_digest IS DISTINCT FROM excluded.value_digest OR work_item_property_value_identities.deleted_at IS NOT NULL
 OR work_item_property_value_identities.project_id<>excluded.project_id OR work_item_property_value_identities.workspace_id<>excluded.workspace_id;
END; $$;
CREATE FUNCTION sync_item_property_value_identities() RETURNS trigger LANGUAGE plpgsql SET search_path=public,pg_catalog AS $$
BEGIN PERFORM sync_property_value_identities(NEW);RETURN NEW;END; $$;
SELECT sync_property_value_identities(i) FROM issues i;
CREATE TRIGGER item_property_value_identities AFTER INSERT OR UPDATE OF custom_properties,project_id,workspace_id,deleted_at ON issues FOR EACH ROW EXECUTE FUNCTION sync_item_property_value_identities();
CREATE FUNCTION advance_property_value_identity_revision() RETURNS trigger LANGUAGE plpgsql SET search_path=public,pg_catalog AS $$
DECLARE target uuid;
BEGIN
 IF TG_OP='DELETE' THEN target:=OLD.issue_id; ELSE target:=NEW.issue_id; END IF;
 UPDATE issue_edit_revisions SET revision=revision+1 WHERE issue_id=target;
 RETURN NULL;
END; $$;
CREATE TRIGGER property_value_identity_revision AFTER INSERT OR UPDATE OR DELETE ON work_item_property_value_identities FOR EACH ROW EXECUTE FUNCTION advance_property_value_identity_revision();
COMMIT;
