BEGIN;
SET LOCAL search_path = public, pg_catalog;
DO $$ BEGIN IF EXISTS(SELECT 1 FROM customers) OR EXISTS(SELECT 1 FROM customer_properties) THEN RAISE EXCEPTION 'cannot discard customer data';END IF;END; $$;
DROP TABLE customer_events,customer_property_values,customer_property_options,customer_properties,customer_work_items,customer_requests,customers,customer_revisions;
DROP TRIGGER customer_workspace_revision ON workspaces;
DROP FUNCTION seed_customer_revision(),advance_customer_revision();
COMMIT;
