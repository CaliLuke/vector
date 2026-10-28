BEGIN;
SET LOCAL search_path = public, pg_catalog;
CREATE TABLE cycle_edit_revisions (cycle_id uuid PRIMARY KEY REFERENCES cycles(id) ON DELETE CASCADE,revision bigint NOT NULL DEFAULT 1 CHECK(revision>0));
INSERT INTO cycle_edit_revisions(cycle_id) SELECT id FROM cycles;
CREATE FUNCTION advance_cycle_edit_revision() RETURNS trigger LANGUAGE plpgsql SET search_path=public,pg_catalog AS $$
BEGIN
 IF TG_OP='INSERT' THEN INSERT INTO cycle_edit_revisions(cycle_id) VALUES(NEW.id);
 ELSE UPDATE cycle_edit_revisions SET revision=revision+1 WHERE cycle_id=NEW.id;
 IF NOT FOUND THEN RAISE EXCEPTION 'cycle revision is missing'; END IF;
 END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER cycle_edit_revision AFTER INSERT OR UPDATE ON cycles FOR EACH ROW EXECUTE FUNCTION advance_cycle_edit_revision();
CREATE TABLE module_edit_revisions (module_id uuid PRIMARY KEY REFERENCES modules(id) ON DELETE CASCADE,revision bigint NOT NULL DEFAULT 1 CHECK(revision>0));
INSERT INTO module_edit_revisions(module_id) SELECT id FROM modules;
CREATE FUNCTION advance_module_edit_revision() RETURNS trigger LANGUAGE plpgsql SET search_path=public,pg_catalog AS $$
BEGIN
 IF TG_OP='INSERT' THEN INSERT INTO module_edit_revisions(module_id) VALUES(NEW.id);
 ELSE UPDATE module_edit_revisions SET revision=revision+1 WHERE module_id=NEW.id;
 IF NOT FOUND THEN RAISE EXCEPTION 'module revision is missing'; END IF;
 END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER module_edit_revision AFTER INSERT OR UPDATE ON modules FOR EACH ROW EXECUTE FUNCTION advance_module_edit_revision();
COMMIT;
