BEGIN;
SET LOCAL search_path = public, pg_catalog;
-- Account-email effects must be retired before removing their admission kind.
UPDATE work_item_effects SET kind='webhook' WHERE kind='email_task' AND payload->>'name'='plane.bgtasks.webhook_task.send_webhook_deactivation_email';
ALTER TABLE work_item_effects DROP CONSTRAINT work_item_effects_kind_check;
ALTER TABLE work_item_effects ADD CONSTRAINT work_item_effects_kind_check CHECK(kind IN('activity','description_version','soft_delete','webhook','asset_storage_delete','asset_storage_metadata','link_preview','asset_storage_upload','page_transaction','page_version'));
DROP TABLE account_email_challenges;
COMMIT;
