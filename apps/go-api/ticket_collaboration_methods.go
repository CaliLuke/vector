package systemapi

import (
	"context"
	api "github.com/CaliLuke/go-plane/ticket_collaboration/gen/ticket_collaboration"
	"io"
)

func (s *ticketCollaborationService) GetAPIPublicAnchorByAnchorCommentsByCommentIDReactions(ctx context.Context, p *api.GetAPIPublicAnchorByAnchorCommentsByCommentIDReactionsPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_public_anchor_by_anchor_comments_by_comment_id_reactions", p, nil)
}

func (s *ticketCollaborationService) PostAPIPublicAnchorByAnchorCommentsByCommentIDReactions(ctx context.Context, p *api.PostAPIPublicAnchorByAnchorCommentsByCommentIDReactionsPayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_public_anchor_by_anchor_comments_by_comment_id_reactions", p, body)
}

func (s *ticketCollaborationService) DeleteAPIPublicAnchorByAnchorCommentsByCommentIDReactionsByReactionCode(ctx context.Context, p *api.DeleteAPIPublicAnchorByAnchorCommentsByCommentIDReactionsByReactionCodePayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_public_anchor_by_anchor_comments_by_comment_id_reactions_by_reaction_code", p, nil)
}

func (s *ticketCollaborationService) GetAPIPublicAnchorByAnchorIssuesByIssueIDComments(ctx context.Context, p *api.GetAPIPublicAnchorByAnchorIssuesByIssueIDCommentsPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_public_anchor_by_anchor_issues_by_issue_id_comments", p, nil)
}

func (s *ticketCollaborationService) PostAPIPublicAnchorByAnchorIssuesByIssueIDComments(ctx context.Context, p *api.PostAPIPublicAnchorByAnchorIssuesByIssueIDCommentsPayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_public_anchor_by_anchor_issues_by_issue_id_comments", p, body)
}

func (s *ticketCollaborationService) DeleteAPIPublicAnchorByAnchorIssuesByIssueIDCommentsByPk(ctx context.Context, p *api.DeleteAPIPublicAnchorByAnchorIssuesByIssueIDCommentsByPkPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_public_anchor_by_anchor_issues_by_issue_id_comments_by_pk", p, nil)
}

func (s *ticketCollaborationService) GetAPIPublicAnchorByAnchorIssuesByIssueIDCommentsByPk(ctx context.Context, p *api.GetAPIPublicAnchorByAnchorIssuesByIssueIDCommentsByPkPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_public_anchor_by_anchor_issues_by_issue_id_comments_by_pk", p, nil)
}

func (s *ticketCollaborationService) PatchAPIPublicAnchorByAnchorIssuesByIssueIDCommentsByPk(ctx context.Context, p *api.PatchAPIPublicAnchorByAnchorIssuesByIssueIDCommentsByPkPayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "patch_api_public_anchor_by_anchor_issues_by_issue_id_comments_by_pk", p, body)
}

func (s *ticketCollaborationService) GetAPIPublicAnchorByAnchorIssuesByIssueIDReactions(ctx context.Context, p *api.GetAPIPublicAnchorByAnchorIssuesByIssueIDReactionsPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_public_anchor_by_anchor_issues_by_issue_id_reactions", p, nil)
}

func (s *ticketCollaborationService) PostAPIPublicAnchorByAnchorIssuesByIssueIDReactions(ctx context.Context, p *api.PostAPIPublicAnchorByAnchorIssuesByIssueIDReactionsPayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_public_anchor_by_anchor_issues_by_issue_id_reactions", p, body)
}

func (s *ticketCollaborationService) DeleteAPIPublicAnchorByAnchorIssuesByIssueIDReactionsByReactionCode(ctx context.Context, p *api.DeleteAPIPublicAnchorByAnchorIssuesByIssueIDReactionsByReactionCodePayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_public_anchor_by_anchor_issues_by_issue_id_reactions_by_reaction_code", p, nil)
}

func (s *ticketCollaborationService) GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDComments(ctx context.Context, p *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments", p, nil)
}

func (s *ticketCollaborationService) PostAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDComments(ctx context.Context, p *api.PostAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsPayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments", p, body)
}

func (s *ticketCollaborationService) DeleteAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsByPk(ctx context.Context, p *api.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsByPkPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments_by_pk", p, nil)
}

func (s *ticketCollaborationService) GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsByPk(ctx context.Context, p *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsByPkPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments_by_pk", p, nil)
}

func (s *ticketCollaborationService) PatchAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsByPk(ctx context.Context, p *api.PatchAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsByPkPayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "patch_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments_by_pk", p, body)
}

func (s *ticketCollaborationService) GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDLinks(ctx context.Context, p *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDLinksPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_links", p, nil)
}

func (s *ticketCollaborationService) PostAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDLinks(ctx context.Context, p *api.PostAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDLinksPayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_links", p, body)
}

func (s *ticketCollaborationService) DeleteAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDLinksByPk(ctx context.Context, p *api.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDLinksByPkPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_links_by_pk", p, nil)
}

func (s *ticketCollaborationService) GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDLinksByPk(ctx context.Context, p *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDLinksByPkPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_links_by_pk", p, nil)
}

func (s *ticketCollaborationService) PatchAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDLinksByPk(ctx context.Context, p *api.PatchAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDLinksByPkPayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "patch_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_links_by_pk", p, body)
}

func (s *ticketCollaborationService) GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDComments(ctx context.Context, p *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDCommentsPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_comments", p, nil)
}

func (s *ticketCollaborationService) PostAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDComments(ctx context.Context, p *api.PostAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDCommentsPayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_comments", p, body)
}

func (s *ticketCollaborationService) DeleteAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDCommentsByPk(ctx context.Context, p *api.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDCommentsByPkPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_comments_by_pk", p, nil)
}

func (s *ticketCollaborationService) GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDCommentsByPk(ctx context.Context, p *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDCommentsByPkPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_comments_by_pk", p, nil)
}

func (s *ticketCollaborationService) PatchAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDCommentsByPk(ctx context.Context, p *api.PatchAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDCommentsByPkPayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "patch_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_comments_by_pk", p, body)
}

func (s *ticketCollaborationService) GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDLinks(ctx context.Context, p *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDLinksPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_links", p, nil)
}

func (s *ticketCollaborationService) PostAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDLinks(ctx context.Context, p *api.PostAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDLinksPayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_links", p, body)
}

func (s *ticketCollaborationService) DeleteAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDLinksByPk(ctx context.Context, p *api.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDLinksByPkPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_links_by_pk", p, nil)
}

func (s *ticketCollaborationService) GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDLinksByPk(ctx context.Context, p *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDLinksByPkPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_links_by_pk", p, nil)
}

func (s *ticketCollaborationService) PatchAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDLinksByPk(ctx context.Context, p *api.PatchAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDLinksByPkPayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "patch_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_links_by_pk", p, body)
}

func (s *ticketCollaborationService) GetAPIWorkspacesBySlugProjectsByProjectIDCommentsByCommentIDReactions(ctx context.Context, p *api.GetAPIWorkspacesBySlugProjectsByProjectIDCommentsByCommentIDReactionsPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_workspaces_by_slug_projects_by_project_id_comments_by_comment_id_reactions", p, nil)
}

func (s *ticketCollaborationService) PostAPIWorkspacesBySlugProjectsByProjectIDCommentsByCommentIDReactions(ctx context.Context, p *api.PostAPIWorkspacesBySlugProjectsByProjectIDCommentsByCommentIDReactionsPayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_workspaces_by_slug_projects_by_project_id_comments_by_comment_id_reactions", p, body)
}

func (s *ticketCollaborationService) DeleteAPIWorkspacesBySlugProjectsByProjectIDCommentsByCommentIDReactionsByReactionCode(ctx context.Context, p *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDCommentsByCommentIDReactionsByReactionCodePayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_workspaces_by_slug_projects_by_project_id_comments_by_comment_id_reactions_by_reaction_code", p, nil)
}

func (s *ticketCollaborationService) GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDComments(ctx context.Context, p *api.GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments", p, nil)
}

func (s *ticketCollaborationService) PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDComments(ctx context.Context, p *api.PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsPayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments", p, body)
}

func (s *ticketCollaborationService) DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsByPk(ctx context.Context, p *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsByPkPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments_by_pk", p, nil)
}

func (s *ticketCollaborationService) GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsByPk(ctx context.Context, p *api.GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsByPkPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments_by_pk", p, nil)
}

func (s *ticketCollaborationService) PatchAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsByPk(ctx context.Context, p *api.PatchAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsByPkPayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "patch_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments_by_pk", p, body)
}

func (s *ticketCollaborationService) PutAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsByPk(ctx context.Context, p *api.PutAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsByPkPayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "put_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments_by_pk", p, body)
}

func (s *ticketCollaborationService) GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueLinks(ctx context.Context, p *api.GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueLinksPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_issue_links", p, nil)
}

func (s *ticketCollaborationService) PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueLinks(ctx context.Context, p *api.PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueLinksPayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_issue_links", p, body)
}

func (s *ticketCollaborationService) DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueLinksByPk(ctx context.Context, p *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueLinksByPkPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_issue_links_by_pk", p, nil)
}

func (s *ticketCollaborationService) GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueLinksByPk(ctx context.Context, p *api.GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueLinksByPkPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_issue_links_by_pk", p, nil)
}

func (s *ticketCollaborationService) PatchAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueLinksByPk(ctx context.Context, p *api.PatchAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueLinksByPkPayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "patch_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_issue_links_by_pk", p, body)
}

func (s *ticketCollaborationService) PutAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueLinksByPk(ctx context.Context, p *api.PutAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueLinksByPkPayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "put_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_issue_links_by_pk", p, body)
}

func (s *ticketCollaborationService) GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueSubscribers(ctx context.Context, p *api.GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueSubscribersPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_issue_subscribers", p, nil)
}

func (s *ticketCollaborationService) PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueSubscribers(ctx context.Context, p *api.PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueSubscribersPayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_issue_subscribers", p, body)
}

func (s *ticketCollaborationService) DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueSubscribersBySubscriberID(ctx context.Context, p *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueSubscribersBySubscriberIDPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_issue_subscribers_by_subscriber_id", p, nil)
}

func (s *ticketCollaborationService) GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDReactions(ctx context.Context, p *api.GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDReactionsPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_reactions", p, nil)
}

func (s *ticketCollaborationService) PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDReactions(ctx context.Context, p *api.PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDReactionsPayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_reactions", p, body)
}

func (s *ticketCollaborationService) DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDReactionsByReactionCode(ctx context.Context, p *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDReactionsByReactionCodePayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_reactions_by_reaction_code", p, nil)
}

func (s *ticketCollaborationService) DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDSubscribe(ctx context.Context, p *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDSubscribePayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_subscribe", p, nil)
}

func (s *ticketCollaborationService) GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDSubscribe(ctx context.Context, p *api.GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDSubscribePayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_subscribe", p, nil)
}

func (s *ticketCollaborationService) PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDSubscribe(ctx context.Context, p *api.PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDSubscribePayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_subscribe", p, body)
}
