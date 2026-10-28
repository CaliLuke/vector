BEGIN;
SET LOCAL search_path = public, pg_catalog;
CREATE TABLE file_asset_edit_revisions (
 asset_id uuid PRIMARY KEY REFERENCES file_assets(id) ON DELETE CASCADE,
 revision bigint NOT NULL DEFAULT 1 CHECK(revision > 0)
);
INSERT INTO file_asset_edit_revisions(asset_id) SELECT id FROM file_assets;
CREATE FUNCTION advance_file_asset_edit_revision() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 INSERT INTO file_asset_edit_revisions(asset_id) VALUES(NEW.id)
 ON CONFLICT(asset_id) DO UPDATE SET revision=file_asset_edit_revisions.revision+1;
 RETURN NEW;
END $$;
CREATE TRIGGER file_asset_edit_revision AFTER INSERT OR UPDATE ON file_assets
 FOR EACH ROW EXECUTE FUNCTION advance_file_asset_edit_revision();

-- Bytes are committed before object creation and erased only after completion.
CREATE TABLE file_asset_upload_intents (
 asset_id uuid PRIMARY KEY REFERENCES file_assets(id) ON DELETE CASCADE,
 object_key text NOT NULL,
 content_type text NOT NULL,
 contents bytea NOT NULL CHECK(octet_length(contents)<=5242880),
 activity jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 completed_at timestamptz
);
-- A staged upload must finish or cancel before its object identity can change.
CREATE FUNCTION guard_pending_file_asset_upload() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF EXISTS(SELECT 1 FROM file_asset_upload_intents WHERE asset_id=OLD.id AND completed_at IS NULL) THEN
  IF TG_OP='DELETE' THEN
   RAISE EXCEPTION 'complete or cancel pending attachment upload before purge';
  ELSIF NEW.asset IS DISTINCT FROM OLD.asset THEN
   RAISE EXCEPTION 'pending attachment upload object key is immutable';
  END IF;
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER pending_file_asset_upload_guard BEFORE UPDATE OF asset OR DELETE ON file_assets
 FOR EACH ROW EXECUTE FUNCTION guard_pending_file_asset_upload();

ALTER TABLE work_item_effects DROP CONSTRAINT work_item_effects_kind_check;
ALTER TABLE work_item_effects ADD CONSTRAINT work_item_effects_kind_check CHECK(kind IN ('activity','description_version','soft_delete','webhook','asset_storage_delete','asset_storage_metadata','link_preview','asset_storage_upload'));
COMMIT;
