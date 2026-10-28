BEGIN;
SET LOCAL search_path = public, pg_catalog;
-- Changes through HTTP, MCP, soft deletion, or maintenance invalidate the same
-- resource revision. Lock counters in identifier order when a relation moves.
CREATE FUNCTION advance_planning_edit_revision() RETURNS trigger LANGUAGE plpgsql SET search_path=public,pg_catalog AS $$
DECLARE
 resource_name text := TG_ARGV[0];
 resource_column text := resource_name || '_id';
 revision_table text := resource_name || '_edit_revisions';
 identifiers uuid[] := ARRAY[]::uuid[];
BEGIN
 IF TG_OP <> 'INSERT' THEN identifiers := array_append(identifiers, (to_jsonb(OLD)->>resource_column)::uuid); END IF;
 IF TG_OP <> 'DELETE' THEN identifiers := array_append(identifiers, (to_jsonb(NEW)->>resource_column)::uuid); END IF;
 EXECUTE format('SELECT %I FROM %I WHERE %I=ANY($1) ORDER BY %I FOR UPDATE', resource_column, revision_table, resource_column, resource_column) USING identifiers;
 EXECUTE format('UPDATE %I SET revision=revision+1 WHERE %I=ANY($1)', revision_table, resource_column) USING identifiers;
 RETURN NULL;
END;
$$;
CREATE TRIGGER cycle_issue_edit_revision AFTER INSERT OR UPDATE OR DELETE ON cycle_issues FOR EACH ROW EXECUTE FUNCTION advance_planning_edit_revision('cycle');
CREATE TRIGGER module_issue_edit_revision AFTER INSERT OR UPDATE OR DELETE ON module_issues FOR EACH ROW EXECUTE FUNCTION advance_planning_edit_revision('module');
CREATE TRIGGER module_member_edit_revision AFTER INSERT OR UPDATE OR DELETE ON module_members FOR EACH ROW EXECUTE FUNCTION advance_planning_edit_revision('module');
COMMIT;
