BEGIN;
SET LOCAL search_path = public, pg_catalog;
-- Retained upload effects keep their discriminator valid across rollback.
-- A rollback must not discard admitted, incomplete object-upload intents.
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM file_asset_upload_intents WHERE completed_at IS NULL) THEN
  RAISE EXCEPTION 'complete pending attachment uploads before rollback';
 END IF;
END $$;
-- Synchronous completion may leave only the worker acknowledgement pending.
UPDATE work_item_effects SET admitted_at=clock_timestamp(),payload='{}'::jsonb,last_error=''
 WHERE kind='asset_storage_upload' AND admitted_at IS NULL
 AND (EXISTS(SELECT 1 FROM file_asset_upload_intents intent WHERE intent.asset_id::text=work_item_effects.payload->>'asset_id' AND intent.completed_at IS NOT NULL)
      OR NOT EXISTS(SELECT 1 FROM file_assets asset WHERE asset.id::text=work_item_effects.payload->>'asset_id'));
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM work_item_effects WHERE kind='asset_storage_upload' AND admitted_at IS NULL) THEN
  RAISE EXCEPTION 'resolve pending attachment upload effects before rollback';
 END IF;
END $$;
DROP TRIGGER pending_file_asset_upload_guard ON file_assets;
DROP FUNCTION guard_pending_file_asset_upload();
DROP TABLE file_asset_upload_intents;
DROP TRIGGER file_asset_edit_revision ON file_assets;
DROP FUNCTION advance_file_asset_edit_revision();
DROP TABLE file_asset_edit_revisions;
COMMIT;
