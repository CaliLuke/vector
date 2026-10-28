package mcpactions

import (
	"context"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/features"
	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/mcptransport"
	"github.com/CaliLuke/go-plane/internal/milestones"
	membership "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	states "github.com/CaliLuke/go-plane/internal/projectstatereads"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	projects "github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
)

// Discovery is a projection of current domain capabilities. Keep the permission
// reads separate from the action catalogue; each invocation still authorizes its
// own target through the native application.
func (s *Service) DiscoverActions(ctx context.Context) (mcptransport.ActionVisibility, error) {
	ctx, err := mcpauth.Revalidate(ctx)
	if err != nil {
		return nil, err
	}
	actor, _ := mcpauth.PrincipalFromContext(ctx)
	if !actor.AllowsAction(false) {
		return mcptransport.ActionVisibility{}, nil
	}
	capabilities, err := s.discoveryCapabilities(ctx, actor)
	if err != nil {
		return nil, err
	}
	return capabilities.actionVisibility(), nil
}

type discoveryCapabilities struct {
	tokenWrite bool
	reads      membership.ReadCapabilities
	types      workitemtypes.APICapabilities
	features   features.Capabilities
	milestones milestones.Capabilities
	states     states.StateCapabilities
	labelWrite bool
	projects   projects.ProjectWriteVisibility
}

func (s *Service) discoveryCapabilities(ctx context.Context, actor mcpauth.Principal) (discoveryCapabilities, error) {
	result := discoveryCapabilities{tokenWrite: actor.AllowsAction(true)}
	var err error
	result.reads, err = s.directory.ReadCapabilities(ctx, uuid.UUID(actor.UserID), uuid.UUID(actor.WorkspaceID), actor.WorkspaceSlug)
	if err != nil {
		return discoveryCapabilities{}, err
	}
	result.types, err = s.types.APIVisibility(ctx, workitemtypes.APIScope{ActorID: actor.UserID, WorkspaceID: actor.WorkspaceID, WorkspaceSlug: actor.WorkspaceSlug})
	if err != nil {
		return discoveryCapabilities{}, err
	}
	if result.reads.Workspace {
		result.features, err = s.features.WorkspaceVisibility(ctx, features.Query{ActorID: actor.UserID, WorkspaceID: actor.WorkspaceID, WorkspaceSlug: actor.WorkspaceSlug})
		if err != nil {
			return discoveryCapabilities{}, err
		}
	}
	result.milestones, err = s.milestones.VisibleActions(ctx, actor.UserID, actor.WorkspaceID, actor.WorkspaceSlug)
	if err != nil {
		return discoveryCapabilities{}, err
	}
	result.states, err = s.states.VisibleActions(ctx, actor.UserID, actor.WorkspaceID, actor.WorkspaceSlug)
	if err != nil {
		return discoveryCapabilities{}, err
	}
	if !result.tokenWrite || !result.reads.Workspace {
		return result, nil
	}
	result.labelWrite, err = s.labelWrites.VisibleLabelWrites(ctx, actor.UserID, actor.WorkspaceID, actor.WorkspaceSlug)
	if err != nil {
		return discoveryCapabilities{}, err
	}
	result.projects, err = s.projectWrites.VisibleProjectWrites(ctx, actor.UserID, actor.WorkspaceID, actor.WorkspaceSlug)
	if err != nil {
		return discoveryCapabilities{}, err
	}
	return result, nil
}

func (capabilities discoveryCapabilities) actionVisibility() mcptransport.ActionVisibility {
	visible := mcptransport.ActionVisibility{}
	reads, typeAccess := capabilities.reads, capabilities.types
	featureAccess, milestone, states := capabilities.features, capabilities.milestones, capabilities.states
	tokenWrite := capabilities.tokenWrite
	workspaceWrite := typeAccess.WorkspaceWrite && tokenWrite
	projectWrite := typeAccess.ProjectWrite && tokenWrite
	projectMemberWrite := reads.ProjectMembers && tokenWrite
	workspaceMemberWrite := reads.Workspace && tokenWrite
	visible["workitem_property"] = map[string]bool{}
	for _, action := range []string{"list", "retrieve", "list_options", "retrieve_option"} {
		visible["workitem_property"][action] = typeAccess.Read
	}
	for _, action := range []string{"create", "update", "delete", "manage_type_properties", "create_option", "update_option", "delete_option"} {
		visible["workitem_property"][action] = workspaceWrite
	}
	visible["workitem_type"] = map[string]bool{"list": typeAccess.Read, "retrieve": typeAccess.Read, "resolve": projectMemberWrite, "import_to_project": projectWrite}
	for _, action := range []string{"create", "update", "delete"} {
		visible["workitem_type"][action] = workspaceWrite
	}
	visible["intake"] = map[string]bool{"list": reads.ProjectMembers, "retrieve": reads.ProjectMembers, "create": projectMemberWrite, "delete": projectMemberWrite, "update": projectWrite}

	for tool, actions := range map[string][]string{"release": {"create", "update", "delete", "update_changelog", "manage_workitems"}, "release_label": {"create", "update", "delete", "attach", "detach"}, "release_tag": {"create", "update", "delete"}} {
		visible[tool] = map[string]bool{"list": typeAccess.Read}
		for _, action := range actions {
			visible[tool][action] = workspaceWrite
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
		visible["page"][action] = workspaceMemberWrite
	}
	for _, action := range []string{"attach_to_workitem", "detach_from_workitem"} {
		visible["page"][action] = projectWrite
	}
	for _, action := range []string{"create", "update", "delete", "add_pages", "remove_page", "add_member", "update_member", "remove_member"} {
		visible["collection"][action] = workspaceMemberWrite
	}
	visible["workspace"] = map[string]bool{"retrieve": true, "get_features": reads.Workspace, "update_features": workspaceWrite}
	visible["template"] = map[string]bool{"list": reads.Workspace}
	for _, action := range []string{"create", "update", "delete"} {
		visible["template"][action] = workspaceWrite || projectWrite
	}

	enabled, customersEnabled := featureAccess.Initiatives, featureAccess.Customers
	customerWrite := featureAccess.ManageCustomers && tokenWrite
	initiativeWrite := featureAccess.ManageInitiatives && tokenWrite

	for tool, actions := range map[string][]string{"customer": {"list", "retrieve", "list_workitems"}, "customer_property": {"list", "retrieve", "get_values"}, "customer_request": {"list", "retrieve"}} {
		visible[tool] = map[string]bool{}
		for _, action := range actions {
			visible[tool][action] = customersEnabled
		}
		for _, action := range []string{"create", "update", "delete"} {
			visible[tool][action] = customerWrite
		}
	}
	visible["customer"]["manage_workitems"], visible["customer_property"]["set_values"] = customerWrite, customerWrite
	visible["initiative"] = map[string]bool{}
	for _, action := range []string{"list", "retrieve", "list_projects", "list_workitems"} {
		visible["initiative"][action] = enabled
	}
	for _, action := range []string{"create", "update", "delete", "add_projects", "remove_projects", "manage_workitems"} {
		visible["initiative"][action] = initiativeWrite
	}
	visible["member"] = map[string]bool{"me": true, "list_workspace": reads.WorkspaceMembers, "list_project": reads.ProjectMembers, "list_roles": reads.Workspace, "retrieve_role": reads.Workspace}
	for _, tool := range []string{"cycle", "module", "label", "workitem_comment", "workitem_link", "workitem_activity"} {
		visible[tool] = map[string]bool{"list": reads.ProjectMembers, "retrieve": reads.ProjectMembers}
	}
	for _, action := range []string{"create", "update", "delete"} {
		visible["workitem_comment"][action] = projectMemberWrite
	}
	visible["project"] = map[string]bool{"list": reads.Workspace, "retrieve": reads.Workspace, "worklog_summary": reads.ProjectMembers, "get_features": reads.ProjectMembers, "update_features": projectWrite}

	write := milestone.Write && tokenWrite
	visible["milestone"] = map[string]bool{"list": milestone.Read, "retrieve": milestone.Read, "list_workitems": milestone.Read, "create": write, "update": write, "delete": write, "manage_workitems": write}

	stateWrite := states.Write && tokenWrite
	visible["workitem_property"]["get_value"] = reads.ProjectMembers
	visible["workitem_property"]["set_value"], visible["workitem_property"]["delete_value"] = stateWrite, stateWrite
	visible["workitem_relation"] = map[string]bool{"list": reads.ProjectMembers, "list_definitions": typeAccess.Read, "create": stateWrite, "delete": stateWrite, "create_definition": workspaceWrite, "update_definition": workspaceWrite, "delete_definition": workspaceWrite}
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
	if !tokenWrite || !reads.Workspace {
		return visible
	}
	labelWrite := capabilities.labelWrite
	for _, action := range []string{"create", "update", "delete"} {
		visible["label"][action] = labelWrite
	}
	writes := capabilities.projects
	visible["project"]["archive"], visible["project"]["unarchive"] = writes.Archive, writes.Unarchive
	visible["project"]["create"] = writes.Archive
	visible["project_estimate"]["link"] = writes.Unarchive
	visible["project"]["update"], visible["project"]["delete"] = writes.Unarchive, writes.Unarchive
	return visible
}
