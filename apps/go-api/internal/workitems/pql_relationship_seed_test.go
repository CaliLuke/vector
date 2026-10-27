//go:build integration

package workitems

// Deliberately malformed legacy references prove that PQL never treats a raw
// foreign key as evidence of a visible relationship, including under negation.
const pqlRelationshipSeed = `
ALTER TABLE issue_types ADD COLUMN workspace_id uuid, ADD COLUMN deleted_at timestamptz, ADD COLUMN is_active bool;
ALTER TABLE issue_labels ADD COLUMN project_id uuid, ADD COLUMN workspace_id uuid;
ALTER TABLE cycle_issues ADD COLUMN project_id uuid, ADD COLUMN workspace_id uuid;
ALTER TABLE module_issues ADD COLUMN project_id uuid, ADD COLUMN workspace_id uuid;
ALTER TABLE cycles ADD COLUMN start_date timestamptz, ADD COLUMN end_date timestamptz, ADD COLUMN archived_at timestamptz;
UPDATE issue_labels SET project_id='00000000-0000-4000-8000-000000000002',workspace_id='00000000-0000-4000-8000-000000000001';
INSERT INTO states(id,workspace_id,project_id,"group",deleted_at) VALUES
 ('00000000-0000-4000-8000-000000000056','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','completed',NULL),
 ('00000000-0000-4000-8000-000000000057','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000099','started',NULL),
 ('00000000-0000-4000-8000-000000000058','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','started',now()),
 ('00000000-0000-4000-8000-000000000059','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','backlog',NULL),
 ('00000000-0000-4000-8000-000000000060','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','unstarted',NULL),
 ('00000000-0000-4000-8000-000000000061','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','cancelled',NULL);
UPDATE issues SET state_id=CASE sequence_id
 WHEN 2 THEN '00000000-0000-4000-8000-000000000056'::uuid
 WHEN 3 THEN '00000000-0000-4000-8000-000000000057'::uuid
 WHEN 4 THEN '00000000-0000-4000-8000-000000000058'::uuid ELSE state_id END;
INSERT INTO issues(id,workspace_id,project_id,name,sequence_id,state_id,target_date,start_date)
SELECT ('90000000-0000-4000-8000-'||lpad(n::text,12,'0'))::uuid,
 '00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','Extra',n,
 ('00000000-0000-4000-8000-'||lpad((n+50)::text,12,'0'))::uuid,
 CASE WHEN n=9 THEN '2026-10-26' ELSE '2026-10-28' END::date,NULL
FROM generate_series(9,11) n;
INSERT INTO issue_types(id,workspace_id,is_epic,is_active,deleted_at) VALUES
 ('70000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000001',true,false,NULL),
 ('70000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000001',false,true,NULL),
 ('70000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000099',true,true,NULL),
 ('70000000-0000-4000-8000-000000000004','00000000-0000-4000-8000-000000000001',true,true,now());
UPDATE issues SET type_id=('70000000-0000-4000-8000-'||lpad(sequence_id::text,12,'0'))::uuid WHERE sequence_id BETWEEN 1 AND 4 OR sequence_id=10;
UPDATE issues SET type_id='70000000-0000-4000-8000-000000000002' WHERE sequence_id=11;
INSERT INTO issue_labels(issue_id,label_id,project_id,workspace_id)
SELECT id,'00000000-0000-4000-8000-000000000006',
 CASE WHEN sequence_id=3 THEN '00000000-0000-4000-8000-000000000099'::uuid ELSE project_id END,
 CASE WHEN sequence_id=4 THEN '00000000-0000-4000-8000-000000000099'::uuid ELSE workspace_id END
FROM issues WHERE sequence_id IN(1,3,4);
INSERT INTO issue_labels(issue_id,label_id,project_id,workspace_id)
SELECT id,'ffffffff-ffff-4fff-8fff-ffffffffffff',project_id,workspace_id FROM issues WHERE sequence_id=9;
INSERT INTO cycles(id,workspace_id,project_id,name,start_date,end_date,archived_at,deleted_at)
SELECT ('60000000-0000-4000-8000-'||lpad(n::text,12,'0'))::uuid,
 CASE WHEN n=9 THEN '00000000-0000-4000-8000-000000000099' ELSE '00000000-0000-4000-8000-000000000001' END::uuid,
 CASE WHEN n=8 THEN '00000000-0000-4000-8000-000000000099' ELSE '00000000-0000-4000-8000-000000000002' END::uuid,
 'Cycle',
 CASE WHEN n=4 THEN NULL WHEN n IN(3,5) THEN '2026-10-28 07:00Z' WHEN n=2 THEN '2026-10-25 07:00Z' ELSE '2026-10-27 16:00Z' END::timestamptz,
 CASE WHEN n=5 THEN NULL WHEN n IN(2,4) THEN '2026-10-27 06:59:59Z' ELSE '2026-10-28 06:59:59Z' END::timestamptz,
 CASE WHEN n=6 THEN now() END,CASE WHEN n=7 THEN now() END
FROM generate_series(1,9) n;
INSERT INTO cycle_issues(issue_id,cycle_id,project_id,workspace_id)
SELECT i.id,c.id,i.project_id,i.workspace_id FROM issues i JOIN cycles c ON
 (i.sequence_id=1 AND c.name='Cycle' AND c.id='60000000-0000-4000-8000-000000000001') OR
 (i.sequence_id=2 AND c.id='60000000-0000-4000-8000-000000000006') OR
 (i.sequence_id=3 AND c.id IN('60000000-0000-4000-8000-000000000007','60000000-0000-4000-8000-000000000008')) OR
 (i.sequence_id=4 AND c.id='60000000-0000-4000-8000-000000000009') OR
 (i.sequence_id=9 AND c.id='60000000-0000-4000-8000-000000000002') OR
 (i.sequence_id=10 AND c.id='60000000-0000-4000-8000-000000000003') OR
 (i.sequence_id=11 AND c.id IN('60000000-0000-4000-8000-000000000004','60000000-0000-4000-8000-000000000005'));
INSERT INTO cycle_issues SELECT * FROM cycle_issues WHERE issue_id='00000000-0000-4000-8000-000000000004';
INSERT INTO cycle_issues(issue_id,cycle_id,project_id,workspace_id,deleted_at)
SELECT id,'60000000-0000-4000-8000-000000000001',
 CASE WHEN sequence_id=3 THEN '00000000-0000-4000-8000-000000000099'::uuid ELSE project_id END,
 CASE WHEN sequence_id=4 THEN '00000000-0000-4000-8000-000000000099'::uuid ELSE workspace_id END,
 CASE WHEN sequence_id=9 THEN now() END
FROM issues WHERE sequence_id IN(3,4,9);
INSERT INTO modules(id,workspace_id,project_id,name,archived_at,deleted_at)
SELECT ('50000000-0000-4000-8000-'||lpad(n::text,12,'0'))::uuid,
 CASE WHEN n=5 THEN '00000000-0000-4000-8000-000000000099' ELSE '00000000-0000-4000-8000-000000000001' END::uuid,
 CASE WHEN n=4 THEN '00000000-0000-4000-8000-000000000099' ELSE '00000000-0000-4000-8000-000000000002' END::uuid,
 'Module',CASE WHEN n=2 THEN now() END,CASE WHEN n=3 THEN now() END
FROM generate_series(1,5) n;
INSERT INTO module_issues(issue_id,module_id,project_id,workspace_id)
SELECT i.id,m.id,i.project_id,i.workspace_id FROM issues i JOIN modules m ON
 m.id=('50000000-0000-4000-8000-'||lpad((CASE WHEN i.sequence_id=9 THEN 5 ELSE i.sequence_id END)::text,12,'0'))::uuid;
INSERT INTO module_issues SELECT * FROM module_issues WHERE issue_id='00000000-0000-4000-8000-000000000004';
INSERT INTO module_issues(issue_id,module_id,project_id,workspace_id,deleted_at)
SELECT id,'50000000-0000-4000-8000-000000000001',
 CASE WHEN sequence_id=10 THEN '00000000-0000-4000-8000-000000000099'::uuid ELSE project_id END,
 CASE WHEN sequence_id=11 THEN '00000000-0000-4000-8000-000000000099'::uuid ELSE workspace_id END,
 CASE WHEN sequence_id=3 THEN now() END
FROM issues WHERE sequence_id IN(3,10,11);
INSERT INTO module_issues(issue_id,module_id,project_id,workspace_id)
SELECT id,'ffffffff-ffff-4fff-8fff-ffffffffffff',project_id,workspace_id FROM issues WHERE sequence_id=4;
INSERT INTO intake_issues(issue_id,intake_id,workspace_id,project_id,status,deleted_at)
SELECT id,'00000000-0000-4000-8000-000000000088',workspace_id,
 CASE WHEN sequence_id=3 THEN '00000000-0000-4000-8000-000000000099'::uuid ELSE project_id END,2,
 CASE WHEN sequence_id=4 THEN now() END
FROM issues WHERE sequence_id IN(2,3,4);
`
