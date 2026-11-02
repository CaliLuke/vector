BEGIN;
SET LOCAL search_path = public, pg_catalog;
CREATE TABLE account_email_challenges (
 user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 current_email text NOT NULL,
 email text NOT NULL,
 code_hash text NOT NULL,
 expires_at timestamptz NOT NULL,
 attempts integer NOT NULL DEFAULT 0 CHECK (attempts BETWEEN 0 AND 5)
);
ALTER TABLE work_item_effects DROP CONSTRAINT work_item_effects_kind_check;
ALTER TABLE work_item_effects ADD CONSTRAINT work_item_effects_kind_check CHECK(kind IN('activity','description_version','soft_delete','webhook','asset_storage_delete','asset_storage_metadata','link_preview','asset_storage_upload','page_transaction','page_version','email_task'));
UPDATE work_item_effects SET kind='email_task' WHERE kind='webhook' AND payload->>'name'='plane.bgtasks.webhook_task.send_webhook_deactivation_email';
COMMIT;
