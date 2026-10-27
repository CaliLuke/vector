//go:build integration

package milestones_test

// The minimal read fixture deliberately admits malformed imported links.
// Bootstrap tests separately prove the canonical schema's ownership constraints.
const milestoneReadSeed = `
CREATE TABLE milestones(id uuid PRIMARY KEY,project_id uuid,workspace_id uuid,title text,target_date date,external_source text,external_id text,created_at timestamptz,updated_at timestamptz,deleted_at timestamptz);
CREATE TABLE milestone_edit_revisions(milestone_id uuid PRIMARY KEY,revision bigint);
CREATE TABLE milestone_issues(id uuid PRIMARY KEY,milestone_id uuid,issue_id uuid,project_id uuid,workspace_id uuid,created_at timestamptz,deleted_at timestamptz);
INSERT INTO milestones
SELECT ('30000000-0000-4000-8000-'||lpad(n::text,12,'0'))::uuid,
 CASE WHEN n=5 THEN '00000000-0000-4000-8000-000000000099' ELSE '00000000-0000-4000-8000-000000000002' END::uuid,
 CASE WHEN n=6 THEN '00000000-0000-4000-8000-000000000099' ELSE '00000000-0000-4000-8000-000000000001' END::uuid,
 CASE n WHEN 1 THEN 'First' WHEN 2 THEN 'Later' ELSE 'Undated' END,
 CASE n WHEN 1 THEN DATE '2026-01-01' WHEN 2 THEN DATE '2026-03-01' END,
 CASE WHEN n=1 THEN ' source ' END,CASE WHEN n=1 THEN 'identity' END,
 '2026-01-01T09:04:05.123456Z','2026-01-01T09:04:05.123456Z',CASE WHEN n=4 THEN now() END
FROM generate_series(1,6) n;
INSERT INTO milestone_edit_revisions SELECT id,7 FROM milestones;
UPDATE issues SET archived_at=now() WHERE sequence_id=2;
INSERT INTO issues(id,project_id,workspace_id,name,sequence_id,is_draft,deleted_at)
SELECT ('50000000-0000-4000-8000-'||lpad(n::text,12,'0'))::uuid,
 CASE WHEN n=5 THEN '00000000-0000-4000-8000-000000000099' ELSE '00000000-0000-4000-8000-000000000002' END::uuid,
 CASE WHEN n=6 THEN '00000000-0000-4000-8000-000000000099' ELSE '00000000-0000-4000-8000-000000000001' END::uuid,
 'Invisible',n,n=3,CASE WHEN n=4 THEN now() END
FROM generate_series(3,6) n;
INSERT INTO milestone_issues
SELECT ('40000000-0000-4000-8000-'||lpad(n::text,12,'0'))::uuid,'30000000-0000-4000-8000-000000000001',
 CASE n WHEN 1 THEN '00000000-0000-4000-8000-000000000004' WHEN 2 THEN '00000000-0000-4000-8000-000000000005' WHEN 8 THEN '00000000-0000-4000-8000-000000000004' WHEN 9 THEN '00000000-0000-4000-8000-000000000004' WHEN 10 THEN '00000000-0000-4000-8000-000000000004' ELSE ('50000000-0000-4000-8000-'||lpad(n::text,12,'0')) END::uuid,
 CASE WHEN n=9 THEN '00000000-0000-4000-8000-000000000099' ELSE '00000000-0000-4000-8000-000000000002' END::uuid,
 CASE WHEN n=10 THEN '00000000-0000-4000-8000-000000000099' ELSE '00000000-0000-4000-8000-000000000001' END::uuid,
 '2026-01-01',CASE WHEN n=8 THEN now() END
FROM generate_series(1,10) n;
`
