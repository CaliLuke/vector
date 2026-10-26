//go:build integration

package testsupport

// EstimateReadSchema retains references to deleted and foreign catalogue rows,
// as can occur before asynchronous cleanup or in legacy data.
const EstimateReadSchema = `
CREATE TABLE workspaces(id uuid PRIMARY KEY,slug text);
CREATE TABLE projects(id uuid PRIMARY KEY,workspace_id uuid,estimate_id uuid,archived_at timestamptz,deleted_at timestamptz);
CREATE TABLE cycles(id uuid PRIMARY KEY,project_id uuid,workspace_id uuid,progress_snapshot jsonb DEFAULT '{}',start_date date DEFAULT CURRENT_DATE-1,end_date date DEFAULT CURRENT_DATE,deleted_at timestamptz);
CREATE TABLE modules(id uuid PRIMARY KEY,project_id uuid,workspace_id uuid,start_date date DEFAULT CURRENT_DATE-1,target_date date DEFAULT CURRENT_DATE);
CREATE TABLE states(id uuid PRIMARY KEY,"group" text);
CREATE TABLE issues(id uuid PRIMARY KEY,project_id uuid,workspace_id uuid,state_id uuid,estimate_point_id uuid,completed_at timestamptz,archived_at timestamptz,deleted_at timestamptz,is_draft bool DEFAULT false);
CREATE TABLE cycle_issues(issue_id uuid,cycle_id uuid,deleted_at timestamptz);
CREATE TABLE module_issues(issue_id uuid,module_id uuid,project_id uuid,workspace_id uuid,deleted_at timestamptz);
CREATE TABLE estimates(id uuid PRIMARY KEY,project_id uuid,workspace_id uuid,type text DEFAULT 'points',deleted_at timestamptz);
CREATE TABLE estimate_points(id uuid PRIMARY KEY,estimate_id uuid,project_id uuid,workspace_id uuid,value text,deleted_at timestamptz);
CREATE TABLE users(id uuid PRIMARY KEY,first_name text,last_name text,display_name text,avatar_asset_id uuid,avatar text,user_timezone text);
CREATE TABLE issue_assignees(issue_id uuid,assignee_id uuid);
CREATE TABLE issue_labels(issue_id uuid,label_id uuid,deleted_at timestamptz);
CREATE TABLE labels(id uuid PRIMARY KEY,project_id uuid,workspace_id uuid,name text,color text,deleted_at timestamptz);
INSERT INTO workspaces VALUES('00000000-0000-4000-8000-000000000001','estimates');
INSERT INTO projects(id,workspace_id) VALUES('00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000001');
INSERT INTO cycles(id,project_id,workspace_id) SELECT '00000000-0000-4000-8000-000000000003',id,workspace_id FROM projects;
INSERT INTO modules(id,project_id,workspace_id) SELECT id,project_id,workspace_id FROM cycles;
INSERT INTO states VALUES('00000000-0000-4000-8000-000000000004','started');
INSERT INTO estimates(id,project_id,workspace_id)
SELECT ('00000000-0000-4000-8000-'||lpad((100+n)::text,12,'0'))::uuid,id,workspace_id FROM projects CROSS JOIN generate_series(1,8) n;
UPDATE projects SET estimate_id='00000000-0000-4000-8000-000000000101';
INSERT INTO estimate_points(id,estimate_id,project_id,workspace_id,value)
SELECT ('00000000-0000-4000-8000-'||lpad((200+n)::text,12,'0'))::uuid,
('00000000-0000-4000-8000-'||lpad((100+n)::text,12,'0'))::uuid,id,workspace_id,
CASE n WHEN 1 THEN '3' WHEN 2 THEN '8' ELSE '100' END FROM projects CROSS JOIN generate_series(1,8) n;
INSERT INTO issues(id,project_id,workspace_id,state_id,estimate_point_id)
SELECT ('00000000-0000-4000-8000-'||lpad((300+n)::text,12,'0'))::uuid,id,workspace_id,
'00000000-0000-4000-8000-000000000004',('00000000-0000-4000-8000-'||lpad((200+n)::text,12,'0'))::uuid
FROM projects CROSS JOIN generate_series(1,8) n;
UPDATE issues SET completed_at=CURRENT_DATE-1 WHERE id='00000000-0000-4000-8000-000000000301';
INSERT INTO cycle_issues(issue_id,cycle_id) SELECT issues.id,cycles.id FROM issues CROSS JOIN cycles;
INSERT INTO module_issues(issue_id,module_id,project_id,workspace_id) SELECT issues.id,modules.id,modules.project_id,modules.workspace_id FROM issues CROSS JOIN modules;
UPDATE estimate_points SET deleted_at=now() WHERE id='00000000-0000-4000-8000-000000000203';
UPDATE estimates SET deleted_at=now() WHERE id='00000000-0000-4000-8000-000000000104';
UPDATE estimate_points SET project_id='00000000-0000-4000-8000-000000000099' WHERE id='00000000-0000-4000-8000-000000000205';
UPDATE estimate_points SET workspace_id='00000000-0000-4000-8000-000000000099' WHERE id='00000000-0000-4000-8000-000000000206';
UPDATE estimates SET project_id='00000000-0000-4000-8000-000000000099' WHERE id='00000000-0000-4000-8000-000000000107';
UPDATE estimates SET workspace_id='00000000-0000-4000-8000-000000000099' WHERE id='00000000-0000-4000-8000-000000000108';
`
