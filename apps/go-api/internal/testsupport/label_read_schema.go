//go:build integration

// Package testsupport holds shared integration data for cross-boundary contracts.
package testsupport

const (
	LabelWorkspace        = "00000000-0000-4000-8000-000000000001"
	LabelProject          = "00000000-0000-4000-8000-000000000002"
	LabelActor            = "00000000-0000-4000-8000-000000000003"
	LabelIssue            = "00000000-0000-4000-8000-000000000004"
	LabelEmptyIssue       = "00000000-0000-4000-8000-000000000005"
	LabelKept             = "00000000-0000-4000-8000-000000000006"
	LabelDeleted          = "00000000-0000-4000-8000-000000000007"
	LabelDraft            = "00000000-0000-4000-8000-000000000011"
	LabelEmptyDraft       = "00000000-0000-4000-8000-000000000012"
	LabelProjectlessDraft = "00000000-0000-4000-8000-000000000013"
	LabelProjectless      = "00000000-0000-4000-8000-000000000014"
	LabelIntake           = "00000000-0000-4000-8000-000000000088"
)

// LabelReadSchema contains the columns selected by current issue projections,
// without product defaults or graph-write behavior. Every test executes real
// production SQL; invalid legacy bridges deliberately remain in the fixture.
const LabelReadSchema = `
CREATE TABLE workspaces(id uuid PRIMARY KEY,slug text,name text,deleted_at timestamptz,created_at timestamptz DEFAULT now());
CREATE TABLE projects(id uuid PRIMARY KEY,workspace_id uuid,name text,identifier text,description text,deleted_at timestamptz,archived_at timestamptz,created_at timestamptz DEFAULT now(),guest_view_all_features bool DEFAULT false);
CREATE TABLE users(id uuid PRIMARY KEY,first_name text,last_name text,display_name text,email text,avatar text,avatar_asset_id uuid,is_bot bool DEFAULT false,is_active bool DEFAULT true,user_timezone text DEFAULT 'UTC',created_at timestamptz DEFAULT now());
CREATE TABLE workspace_members(id uuid,workspace_id uuid,member_id uuid,role int,is_active bool,deleted_at timestamptz,created_at timestamptz DEFAULT now());
CREATE TABLE project_members(id uuid,workspace_id uuid,project_id uuid,member_id uuid,role int,is_active bool,deleted_at timestamptz,created_at timestamptz DEFAULT now());
CREATE TABLE states(id uuid PRIMARY KEY,project_id uuid,workspace_id uuid,name text,"group" text,color text,sequence int,is_triage bool DEFAULT false,deleted_at timestamptz);
CREATE TABLE issues(id uuid PRIMARY KEY,workspace_id uuid,project_id uuid,state_id uuid,parent_id uuid,type_id uuid,estimate_point_id uuid,point numeric,name text,description_html text,description_json jsonb,description_stripped text,description_binary bytea,priority text DEFAULT 'low',start_date date,target_date date,sequence_id int,sort_order double precision DEFAULT 0,completed_at timestamptz,archived_at timestamptz,is_draft bool DEFAULT false,external_source text,external_id text,created_at timestamptz DEFAULT now(),updated_at timestamptz DEFAULT now(),created_by_id uuid,updated_by_id uuid,deleted_at timestamptz,custom_properties jsonb DEFAULT '{}');
CREATE TABLE draft_issues(LIKE issues INCLUDING ALL);
ALTER TABLE draft_issues ALTER COLUMN is_draft SET DEFAULT true;
CREATE TABLE labels(id uuid PRIMARY KEY,workspace_id uuid,project_id uuid,parent_id uuid,name text,color text,sort_order double precision DEFAULT 0,created_at timestamptz DEFAULT now(),updated_at timestamptz DEFAULT now(),deleted_at timestamptz);
CREATE TABLE issue_labels(id uuid,issue_id uuid,label_id uuid,created_at timestamptz DEFAULT now(),deleted_at timestamptz);
CREATE TABLE draft_issue_labels(id uuid,draft_issue_id uuid,label_id uuid,created_at timestamptz DEFAULT now(),deleted_at timestamptz);
CREATE TABLE issue_assignees(id uuid,issue_id uuid,assignee_id uuid,created_at timestamptz DEFAULT now(),deleted_at timestamptz);
CREATE TABLE issue_subscribers(id uuid,issue_id uuid,subscriber_id uuid,created_at timestamptz DEFAULT now(),deleted_at timestamptz);
CREATE TABLE draft_issue_assignees(id uuid,draft_issue_id uuid,assignee_id uuid,created_at timestamptz DEFAULT now(),deleted_at timestamptz);
CREATE TABLE modules(id uuid PRIMARY KEY,workspace_id uuid,project_id uuid,name text,archived_at timestamptz,deleted_at timestamptz);
CREATE TABLE module_issues(id uuid,issue_id uuid,module_id uuid,created_at timestamptz DEFAULT now(),deleted_at timestamptz);
CREATE TABLE draft_issue_modules(id uuid,draft_issue_id uuid,module_id uuid,created_at timestamptz DEFAULT now(),deleted_at timestamptz);
CREATE TABLE cycles(id uuid PRIMARY KEY,workspace_id uuid,project_id uuid,name text,deleted_at timestamptz);
CREATE TABLE cycle_issues(id uuid,issue_id uuid,cycle_id uuid,created_at timestamptz DEFAULT now(),deleted_at timestamptz);
CREATE TABLE draft_issue_cycles(id uuid,draft_issue_id uuid,cycle_id uuid,created_at timestamptz DEFAULT now(),deleted_at timestamptz);
CREATE TABLE estimate_points(id uuid,value numeric);
CREATE TABLE file_assets(id uuid,issue_id uuid,entity_type text,deleted_at timestamptz);
CREATE TABLE issue_links(id uuid,issue_id uuid,url text,title text,created_at timestamptz DEFAULT now(),deleted_at timestamptz);
CREATE TABLE issue_votes(id uuid,issue_id uuid,vote int,actor_id uuid,deleted_at timestamptz);
CREATE TABLE issue_reactions(id uuid,issue_id uuid,reaction text,actor_id uuid,deleted_at timestamptz);
CREATE TABLE issue_comments(id uuid,issue_id uuid,actor_id uuid,comment_html text,comment_stripped text,created_at timestamptz DEFAULT now(),deleted_at timestamptz);
CREATE TABLE issue_relations(id uuid,issue_id uuid,related_issue_id uuid,relation_type text,deleted_at timestamptz);
CREATE TABLE intakes(id uuid PRIMARY KEY,workspace_id uuid,project_id uuid,name text,description text,is_default bool,view_props jsonb,logo_props jsonb,created_by_id uuid,updated_by_id uuid,created_at timestamptz DEFAULT now(),updated_at timestamptz DEFAULT now(),deleted_at timestamptz);
CREATE TABLE intake_issues(id uuid,intake_id uuid,workspace_id uuid,project_id uuid,issue_id uuid,status int,duplicate_to_id uuid,snoozed_till timestamptz,source text,created_by_id uuid,updated_by_id uuid,created_at timestamptz DEFAULT now(),updated_at timestamptz DEFAULT now(),deleted_at timestamptz);
CREATE TABLE exporters(id uuid,token text,type text,initiated_by_id uuid);
`

const LabelReadSeed = `
INSERT INTO workspaces(id,slug,name) VALUES('00000000-0000-4000-8000-000000000001','demo','Demo'),('00000000-0000-4000-8000-000000000099','foreign','Foreign');
INSERT INTO projects(id,workspace_id,name,identifier) VALUES('00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000001','Current','CUR'),('00000000-0000-4000-8000-000000000099','00000000-0000-4000-8000-000000000001','Foreign','FOR');
INSERT INTO users(id,first_name,last_name,display_name) VALUES('00000000-0000-4000-8000-000000000003','Read','Actor','Reader');
INSERT INTO workspace_members(workspace_id,member_id,role,is_active) VALUES('00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000003',15,true);
INSERT INTO project_members(workspace_id,project_id,member_id,role,is_active) VALUES('00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000003',15,true);
INSERT INTO states(id,project_id,workspace_id,name,"group",color) VALUES('00000000-0000-4000-8000-000000000055','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000001','Started','started','#123456');
INSERT INTO issues(id,workspace_id,project_id,state_id,name,sequence_id,created_by_id) VALUES
 ('00000000-0000-4000-8000-000000000004','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000055','Mixed labels',1,'00000000-0000-4000-8000-000000000003'),
 ('00000000-0000-4000-8000-000000000005','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000055','No live labels',2,'00000000-0000-4000-8000-000000000003');
INSERT INTO draft_issues(id,workspace_id,project_id,name,created_by_id) VALUES
 ('00000000-0000-4000-8000-000000000011','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','Draft mixed','00000000-0000-4000-8000-000000000003'),
 ('00000000-0000-4000-8000-000000000012','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','Draft empty','00000000-0000-4000-8000-000000000003'),
 ('00000000-0000-4000-8000-000000000013','00000000-0000-4000-8000-000000000001',NULL,'Projectless','00000000-0000-4000-8000-000000000003');
INSERT INTO labels(id,workspace_id,project_id,name,deleted_at) VALUES
 ('00000000-0000-4000-8000-000000000006','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','Kept',NULL),
 ('00000000-0000-4000-8000-000000000007','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','AAA Deleted',now()),
 ('00000000-0000-4000-8000-000000000008','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','Removed bridge',NULL),
 ('00000000-0000-4000-8000-000000000009','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000099','Foreign project',NULL),
 ('00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000099','00000000-0000-4000-8000-000000000002','Foreign workspace',NULL),
 ('00000000-0000-4000-8000-000000000014','00000000-0000-4000-8000-000000000001',NULL,'Projectless label',NULL);
INSERT INTO issue_labels(issue_id,label_id,deleted_at) SELECT '00000000-0000-4000-8000-000000000004',id,CASE WHEN name='Removed bridge' THEN now() END FROM labels;
INSERT INTO issue_labels(issue_id,label_id) VALUES('00000000-0000-4000-8000-000000000005','00000000-0000-4000-8000-000000000007');
INSERT INTO draft_issue_labels(draft_issue_id,label_id,deleted_at) SELECT '00000000-0000-4000-8000-000000000011',label_id,deleted_at FROM issue_labels WHERE issue_id='00000000-0000-4000-8000-000000000004';
INSERT INTO draft_issue_labels(draft_issue_id,label_id) VALUES('00000000-0000-4000-8000-000000000012','00000000-0000-4000-8000-000000000007'),('00000000-0000-4000-8000-000000000013','00000000-0000-4000-8000-000000000014'),('00000000-0000-4000-8000-000000000013','00000000-0000-4000-8000-000000000006');
INSERT INTO intakes(id,workspace_id,project_id,name) VALUES('00000000-0000-4000-8000-000000000088','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','Intake');
INSERT INTO intake_issues(id,intake_id,workspace_id,project_id,issue_id,status,source,created_by_id) VALUES('00000000-0000-4000-8000-000000000089','00000000-0000-4000-8000-000000000088','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000004',0,'IN_APP','00000000-0000-4000-8000-000000000003');
`
