//go:build integration

package testsupport

// MilestoneWriteSeed supplies a valid actor and project for native writer and worker proofs.
const MilestoneWriteSeed = `
INSERT INTO users(id,username,email,password,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,token,user_timezone,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,display_name,is_email_valid,is_password_reset_required)
VALUES('00000000-0000-4000-8000-000000000003','writer','writer@example.test','!','','','',now(),now(),now(),'','',false,false,false,true,false,true,false,'','UTC','','','','',false,'Writer',true,false);
INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at)
VALUES('00000000-0000-4000-8000-000000000001','Milestones','00000000-0000-4000-8000-000000000003','milestones','UTC','',now(),now());
INSERT INTO workspace_members(id,workspace_id,member_id,role,created_at,updated_at,view_props,default_props,issue_props,is_active,getting_started_checklist,tips,explored_features)
VALUES(gen_random_uuid(),'00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000003',15,now(),now(),'{}','{}','{}',true,'{}','{}','{}');
INSERT INTO projects(id,name,description,network,workspace_id,identifier,module_view,cycle_view,issue_views_view,page_view,intake_view,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,archive_in,close_in,logo_props,timezone,created_at,updated_at)
VALUES('00000000-0000-4000-8000-000000000002','Milestones','',0,'00000000-0000-4000-8000-000000000001','MILE',false,false,false,true,false,false,false,false,0,0,'{}','UTC',now(),now());
INSERT INTO project_members(id,workspace_id,project_id,member_id,role,created_at,updated_at,view_props,default_props,sort_order,preferences,is_active)
VALUES(gen_random_uuid(),'00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000003',15,now(),now(),'{}','{}',0,'{}',true);
`
