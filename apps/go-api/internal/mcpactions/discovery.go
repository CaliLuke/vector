package mcpactions

import (
	"context"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/features"

	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/mcptransport"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
)

func (s *Service) DiscoverActions(ctx context.Context) (mcptransport.ActionVisibility, error) {
	ctx, err := mcpauth.Revalidate(ctx)
	if err != nil {
		return nil, err
	}
	visible := mcptransport.ActionVisibility{}
	actor, _ := mcpauth.PrincipalFromContext(ctx) // Revalidate returned a verified context.
	if !actor.AllowsAction(false) {
		return visible, nil
	}
	reads, err := s.directory.ReadCapabilities(ctx, uuid.UUID(actor.UserID), uuid.UUID(actor.WorkspaceID), actor.WorkspaceSlug)
	if err != nil {
		return nil, err
	}
	typeAccess, err := s.types.APIVisibility(ctx, workitemtypes.APIScope{ActorID: actor.UserID, WorkspaceID: actor.WorkspaceID, WorkspaceSlug: actor.WorkspaceSlug})
	if err != nil {
		return nil, err
	}
	visible["workitem_property"] = map[string]bool{}
	for _, action := range []string{"list", "retrieve", "list_options", "retrieve_option"} {
		visible["workitem_property"][action] = typeAccess.Read
	}
	for _, action := range []string{"create", "update", "delete", "manage_type_properties", "create_option", "update_option", "delete_option"} {
		visible["workitem_property"][action] = typeAccess.WorkspaceWrite && actor.AllowsAction(true)
	}
	visible["workitem_type"] = map[string]bool{"list": typeAccess.Read, "retrieve": typeAccess.Read, "resolve": reads.ProjectMembers && actor.AllowsAction(true), "import_to_project": typeAccess.ProjectWrite && actor.AllowsAction(true)}
	for _, action := range []string{"create", "update", "delete"} {
		visible["workitem_type"][action] = typeAccess.WorkspaceWrite && actor.AllowsAction(true)
	}
	visible["intake"] = map[string]bool{"list": reads.ProjectMembers, "retrieve": reads.ProjectMembers, "create": reads.ProjectMembers && actor.AllowsAction(true), "delete": reads.ProjectMembers && actor.AllowsAction(true), "update": typeAccess.ProjectWrite && actor.AllowsAction(true)}

	for tool, actions := range map[string][]string{"release": {"create", "update", "delete", "update_changelog", "manage_workitems"}, "release_label": {"create", "update", "delete", "attach", "detach"}, "release_tag": {"create", "update", "delete"}} {
		visible[tool] = map[string]bool{"list": typeAccess.Read}
		for _, action := range actions {
			visible[tool][action] = typeAccess.WorkspaceWrite && actor.AllowsAction(true)
		}
	}
	visible["release"]["retrieve"], visible["release"]["get_changelog"], visible["release"]["list_workitems"], visible["release_tag"]["retrieve"] = typeAccess.Read, typeAccess.Read, typeAccess.Read, typeAccess.Read
	visible["page"] = map[string]bool{"list": reads.Workspace, "retrieve": reads.Workspace, "list_workitem_pages": reads.ProjectMembers}
	visible["collection"] = map[string]bool{}
	for _, action := range []string{"list", "retrieve", "list_pages", "search_pages", "list_members"} {
		visible["collection"][action] = reads.Workspace
	}
	// Target ownership and collection membership are checked by the native application.
	for _, action := range []string{"create", "update", "archive", "delete", "set_collection"} {
		visible["page"][action] = reads.Workspace && actor.AllowsAction(true)
	}
	for _, action := range []string{"attach_to_workitem", "detach_from_workitem"} {
		visible["page"][action] = typeAccess.ProjectWrite && actor.AllowsAction(true)
	}
	for _, action := range []string{"create", "update", "delete", "add_pages", "remove_page", "add_member", "update_member", "remove_member"} {
		visible["collection"][action] = reads.Workspace && actor.AllowsAction(true)
	}
	visible["workspace"] = map[string]bool{"retrieve": true, "get_features": reads.Workspace, "update_features": typeAccess.WorkspaceWrite && actor.AllowsAction(true)}
	visible["template"] = map[string]bool{"list": reads.Workspace}
	for _, action := range []string{"create", "update", "delete"} {
		visible["template"][action] = (typeAccess.WorkspaceWrite || typeAccess.ProjectWrite) && actor.AllowsAction(true)
	}

	featureAccess := features.Capabilities{}
	if reads.Workspace {
		featureAccess, err = s.features.WorkspaceVisibility(ctx, features.Query{ActorID: actor.UserID, WorkspaceID: actor.WorkspaceID, WorkspaceSlug: actor.WorkspaceSlug})
		if err != nil {
			return nil, err
		}
	}
	enabled, customersEnabled := featureAccess.Initiatives, featureAccess.Customers

	for tool, actions := range map[string][]string{"customer": {"list", "retrieve", "list_workitems"}, "customer_property": {"list", "retrieve", "get_values"}, "customer_request": {"list", "retrieve"}} {
		visible[tool] = map[string]bool{}
		for _, action := range actions {
			visible[tool][action] = customersEnabled
		}
		for _, action := range []string{"create", "update", "delete"} {
			visible[tool][action] = featureAccess.ManageCustomers && actor.AllowsAction(true)
		}
	}
	visible["customer"]["manage_workitems"], visible["customer_property"]["set_values"] = featureAccess.ManageCustomers && actor.AllowsAction(true), featureAccess.ManageCustomers && actor.AllowsAction(true)
	visible["initiative"] = map[string]bool{}
	for _, action := range []string{"list", "retrieve", "list_projects", "list_workitems"} {
		visible["initiative"][action] = enabled
	}
	for _, action := range []string{"create", "update", "delete", "add_projects", "remove_projects", "manage_workitems"} {
		visible["initiative"][action] = featureAccess.ManageInitiatives && actor.AllowsAction(true)
	}
	visible["member"] = map[string]bool{"me": true, "list_workspace": reads.WorkspaceMembers, "list_project": reads.ProjectMembers, "list_roles": reads.Workspace, "retrieve_role": reads.Workspace}
	for _, tool := range []string{"cycle", "module", "label", "workitem_comment", "workitem_link", "workitem_activity"} {
		visible[tool] = map[string]bool{"list": reads.ProjectMembers, "retrieve": reads.ProjectMembers}
	}
	for _, action := range []string{"create", "update", "delete"} {
		visible["workitem_comment"][action] = reads.ProjectMembers && actor.AllowsAction(true)
	}
	visible["project"] = map[string]bool{"list": reads.Workspace, "retrieve": reads.Workspace, "worklog_summary": reads.ProjectMembers, "get_features": reads.ProjectMembers, "update_features": typeAccess.ProjectWrite && actor.AllowsAction(true)}
	milestone, err := s.milestones.VisibleActions(ctx, actor.UserID, actor.WorkspaceID, actor.WorkspaceSlug)
	if err != nil {
		return nil, err
	}
	write := milestone.Write && actor.AllowsAction(true)
	visible["milestone"] = map[string]bool{"list": milestone.Read, "retrieve": milestone.Read, "list_workitems": milestone.Read, "create": write, "update": write, "delete": write, "manage_workitems": write}
	states, err := s.states.VisibleActions(ctx, actor.UserID, actor.WorkspaceID, actor.WorkspaceSlug)
	if err != nil {
		return nil, err
	}
	stateWrite := states.Write && actor.AllowsAction(true)
	visible["workitem_property"]["get_value"] = reads.ProjectMembers
	visible["workitem_property"]["set_value"], visible["workitem_property"]["delete_value"] = stateWrite, stateWrite
	visible["workitem_relation"] = map[string]bool{"list": reads.ProjectMembers, "list_definitions": typeAccess.Read, "create": stateWrite, "delete": stateWrite, "create_definition": typeAccess.WorkspaceWrite && actor.AllowsAction(true), "update_definition": typeAccess.WorkspaceWrite && actor.AllowsAction(true), "delete_definition": typeAccess.WorkspaceWrite && actor.AllowsAction(true)}
	visible["work_log"] = map[string]bool{"list": reads.ProjectMembers, "create": stateWrite, "update": stateWrite, "delete": stateWrite}
	visible["workitem_attachment"] = map[string]bool{"list": reads.ProjectMembers, "read": reads.ProjectMembers, "download_url": reads.ProjectMembers, "upload_from_url": stateWrite, "delete": stateWrite}
	visible["get_pql_reference"] = map[string]bool{"": true}
	visible["workitem"] = map[string]bool{"count": reads.Workspace, "list": reads.Workspace, "list_archived": reads.ProjectMembers, "retrieve": reads.ProjectMembers, "retrieve_by_identifier": reads.Workspace, "search": reads.Workspace}
	visible["cycle"]["list_workitems"], visible["module"]["list_workitems"] = reads.ProjectMembers, reads.ProjectMembers
	for _, action := range []string{"create", "update", "delete", "archive", "manage_assignee", "manage_label"} {
		visible["workitem"][action] = stateWrite
	}
	visible["project_estimate"] = map[string]bool{"retrieve": reads.ProjectMembers, "list_points": reads.ProjectMembers}
	for _, action := range []string{"create", "update", "delete", "create_points", "update_point", "delete_point"} {
		visible["project_estimate"][action] = stateWrite
	}
	for _, tool := range []string{"cycle", "module"} {
		for _, action := range []string{"archive", "unarchive", "create", "update", "delete", "manage_workitems"} {
			visible[tool][action] = stateWrite
		}
	}
	for _, action := range []string{"create", "update", "delete"} {
		visible["workitem_link"][action] = stateWrite
	}
	visible["cycle"]["complete"] = stateWrite
	visible["cycle"]["transfer_workitems"] = stateWrite
	visible["state"] = map[string]bool{"list": states.Read, "retrieve": states.Read, "create": stateWrite, "update": stateWrite, "delete": stateWrite}
	if !actor.AllowsAction(true) || !reads.Workspace {
		return visible, nil
	}
	labelWrite, err := s.labelWrites.VisibleLabelWrites(ctx, actor.UserID, actor.WorkspaceID, actor.WorkspaceSlug)
	if err != nil {
		return nil, err
	}
	for _, action := range []string{"create", "update", "delete"} {
		visible["label"][action] = labelWrite
	}
	writes, err := s.projectWrites.VisibleProjectWrites(ctx, actor.UserID, actor.WorkspaceID, actor.WorkspaceSlug)
	if err != nil {
		return nil, err
	}
	visible["project"]["archive"], visible["project"]["unarchive"] = writes.Archive, writes.Unarchive
	visible["project"]["create"] = writes.Archive
	visible["project_estimate"]["link"] = writes.Unarchive
	visible["project"]["update"], visible["project"]["delete"] = writes.Unarchive, writes.Unarchive
	return visible, nil
}
