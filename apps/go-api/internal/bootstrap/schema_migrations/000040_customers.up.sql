BEGIN;
SET LOCAL search_path = public, pg_catalog;
CREATE TABLE customers(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 name varchar(255) NOT NULL CHECK(length(btrim(name))>0),description jsonb NOT NULL DEFAULT '{}',description_html text,description_stripped text,description_binary text,
 email varchar(255),website_url text,logo_props jsonb NOT NULL DEFAULT '{}',domain varchar(255),employees bigint CHECK(employees>=0),stage varchar(255),contract_status varchar(255),revenue varchar(255),
 external_source varchar(255),external_id varchar(255),archived_at timestamptz,created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),deleted_at timestamptz,
 created_by_id uuid REFERENCES users(id) ON DELETE SET NULL,updated_by_id uuid REFERENCES users(id) ON DELETE SET NULL,UNIQUE(id,workspace_id)
);
CREATE UNIQUE INDEX customer_external_identity ON customers(workspace_id,external_source,external_id) WHERE deleted_at IS NULL AND external_source IS NOT NULL AND external_id IS NOT NULL;
CREATE INDEX customer_names ON customers(workspace_id,name) WHERE deleted_at IS NULL;
CREATE TABLE customer_requests(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),workspace_id uuid NOT NULL,customer_id uuid NOT NULL,name varchar(255) NOT NULL CHECK(length(btrim(name))>0),
 description jsonb NOT NULL DEFAULT '{}',description_html text,description_stripped text,link text,
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),deleted_at timestamptz,created_by_id uuid REFERENCES users(id) ON DELETE SET NULL,updated_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 FOREIGN KEY(customer_id,workspace_id) REFERENCES customers(id,workspace_id) ON DELETE CASCADE,UNIQUE(id,customer_id,workspace_id)
);
CREATE TABLE customer_work_items(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),workspace_id uuid NOT NULL,customer_id uuid NOT NULL,request_id uuid,issue_id uuid NOT NULL,project_id uuid NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),created_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 FOREIGN KEY(customer_id,workspace_id) REFERENCES customers(id,workspace_id) ON DELETE CASCADE,
 FOREIGN KEY(request_id,customer_id,workspace_id) REFERENCES customer_requests(id,customer_id,workspace_id) ON DELETE CASCADE,
 FOREIGN KEY(issue_id,project_id,workspace_id) REFERENCES issues(id,project_id,workspace_id) ON UPDATE CASCADE ON DELETE CASCADE,
 UNIQUE NULLS NOT DISTINCT(customer_id,request_id,issue_id)
);
CREATE TABLE customer_properties(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,name varchar(255) NOT NULL,display_name varchar(255) NOT NULL CHECK(length(btrim(display_name))>0),description text,
 logo_props jsonb NOT NULL DEFAULT '{}',sort_order double precision NOT NULL DEFAULT 65535,property_type text NOT NULL CHECK(property_type IN('TEXT','DATETIME','DECIMAL','BOOLEAN','OPTION','RELATION','URL','EMAIL','FILE')),
 relation_type text CHECK(relation_type IN('ISSUE','USER')),is_required boolean NOT NULL DEFAULT false,default_value jsonb NOT NULL DEFAULT '[]' CHECK(jsonb_typeof(default_value)='array'),settings jsonb NOT NULL DEFAULT '{}',is_active boolean NOT NULL DEFAULT true,is_multi boolean NOT NULL DEFAULT false,validation_rules jsonb,
 external_source varchar(255),external_id varchar(255),created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),deleted_at timestamptz,created_by_id uuid REFERENCES users(id) ON DELETE SET NULL,updated_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 UNIQUE(id,workspace_id),CHECK(property_type<>'RELATION' OR relation_type IS NOT NULL)
);
CREATE UNIQUE INDEX customer_property_name ON customer_properties(workspace_id,name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX customer_property_display_name ON customer_properties(workspace_id,display_name) WHERE deleted_at IS NULL;
CREATE TABLE customer_property_options(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),workspace_id uuid NOT NULL,property_id uuid NOT NULL,name varchar(255) NOT NULL CHECK(length(btrim(name))>0),description text,is_default boolean NOT NULL DEFAULT false,is_active boolean NOT NULL DEFAULT true,sort_order double precision NOT NULL DEFAULT 65535,
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),deleted_at timestamptz,created_by_id uuid REFERENCES users(id) ON DELETE SET NULL,updated_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 FOREIGN KEY(property_id,workspace_id) REFERENCES customer_properties(id,workspace_id) ON DELETE CASCADE,UNIQUE(id,property_id,workspace_id)
);
CREATE UNIQUE INDEX customer_option_name ON customer_property_options(property_id,name) WHERE deleted_at IS NULL;
CREATE TABLE customer_property_values(
 customer_id uuid NOT NULL,property_id uuid NOT NULL,workspace_id uuid NOT NULL,value_strings jsonb NOT NULL CHECK(jsonb_typeof(value_strings)='array'),updated_at timestamptz NOT NULL DEFAULT now(),updated_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 PRIMARY KEY(customer_id,property_id),FOREIGN KEY(customer_id,workspace_id) REFERENCES customers(id,workspace_id) ON DELETE CASCADE,FOREIGN KEY(property_id,workspace_id) REFERENCES customer_properties(id,workspace_id) ON DELETE CASCADE
);
CREATE TABLE customer_events(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,resource_id uuid NOT NULL,resource_kind text NOT NULL,action text NOT NULL,
 actor_id uuid REFERENCES users(id) ON DELETE SET NULL,input jsonb NOT NULL,before_value jsonb,after_value jsonb,created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE customer_revisions(workspace_id uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,revision bigint NOT NULL DEFAULT 1 CHECK(revision>0));
INSERT INTO customer_revisions(workspace_id) SELECT id FROM workspaces;
CREATE FUNCTION seed_customer_revision() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN INSERT INTO customer_revisions(workspace_id) VALUES(NEW.id);RETURN NEW;END; $$;
CREATE TRIGGER customer_workspace_revision AFTER INSERT ON workspaces FOR EACH ROW EXECUTE FUNCTION seed_customer_revision();
CREATE FUNCTION advance_customer_revision() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE old_workspace uuid;new_workspace uuid;
BEGIN IF TG_OP<>'INSERT' THEN old_workspace:=OLD.workspace_id;END IF;IF TG_OP<>'DELETE' THEN new_workspace:=NEW.workspace_id;END IF;
 PERFORM workspace_id FROM customer_revisions WHERE workspace_id IN(old_workspace,new_workspace) ORDER BY workspace_id FOR UPDATE;
 UPDATE customer_revisions SET revision=revision+1 WHERE workspace_id IN(old_workspace,new_workspace);RETURN NULL;END; $$;
CREATE TRIGGER customer_revision AFTER INSERT OR UPDATE OR DELETE ON customers FOR EACH ROW EXECUTE FUNCTION advance_customer_revision();
CREATE TRIGGER customer_request_revision AFTER INSERT OR UPDATE OR DELETE ON customer_requests FOR EACH ROW EXECUTE FUNCTION advance_customer_revision();
CREATE TRIGGER customer_item_revision AFTER INSERT OR UPDATE OR DELETE ON customer_work_items FOR EACH ROW EXECUTE FUNCTION advance_customer_revision();
CREATE TRIGGER customer_property_revision AFTER INSERT OR UPDATE OR DELETE ON customer_properties FOR EACH ROW EXECUTE FUNCTION advance_customer_revision();
CREATE TRIGGER customer_option_revision AFTER INSERT OR UPDATE OR DELETE ON customer_property_options FOR EACH ROW EXECUTE FUNCTION advance_customer_revision();
CREATE TRIGGER customer_value_revision AFTER INSERT OR UPDATE OR DELETE ON customer_property_values FOR EACH ROW EXECUTE FUNCTION advance_customer_revision();
CREATE TRIGGER customer_item_edit_revision AFTER INSERT OR UPDATE OR DELETE ON customer_work_items FOR EACH ROW EXECUTE FUNCTION advance_issue_assignment_revision();
COMMIT;
