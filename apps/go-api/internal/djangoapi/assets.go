package djangoapi

func FileAssetURLSQL(assetID, legacy string) string {
	return `CASE WHEN ` + assetID + ` IS NULL THEN NULLIF(` + legacy + `,'') ELSE (SELECT CASE
WHEN asset.entity_type IN ('WORKSPACE_LOGO','USER_AVATAR','USER_COVER','PROJECT_COVER') THEN '/api/assets/v2/static/'||asset.id::text||'/'
WHEN asset.entity_type='ISSUE_ATTACHMENT' THEN '/api/assets/v2/workspaces/'||asset_workspace.slug||'/projects/'||COALESCE(asset.project_id::text,'None')||'/issues/'||COALESCE(asset.issue_id::text,'None')||'/attachments/'||asset.id::text||'/'
WHEN asset.entity_type IN ('ISSUE_DESCRIPTION','COMMENT_DESCRIPTION','PAGE_DESCRIPTION','DRAFT_ISSUE_DESCRIPTION') THEN '/api/assets/v2/workspaces/'||asset_workspace.slug||'/projects/'||COALESCE(asset.project_id::text,'None')||'/'||asset.id::text||'/'
ELSE NULL END FROM file_assets asset LEFT JOIN workspaces asset_workspace ON asset_workspace.id=asset.workspace_id WHERE asset.id=` + assetID + `) END`
}

// FileAssetURLFieldSQL preserves DRF omission when the asset URL property raises
// AttributeError for a missing workspace, while retaining explicit JSON null.
func FileAssetURLFieldSQL(assetID, legacy, field string) string {
	return `CASE WHEN EXISTS(SELECT 1 FROM file_assets missing_asset WHERE missing_asset.id=` + assetID + ` AND missing_asset.entity_type IN ('ISSUE_ATTACHMENT','ISSUE_DESCRIPTION','COMMENT_DESCRIPTION','PAGE_DESCRIPTION','DRAFT_ISSUE_DESCRIPTION') AND missing_asset.workspace_id IS NULL) THEN '{}'::jsonb ELSE jsonb_build_object('` + field + `',` + FileAssetURLSQL(assetID, legacy) + `) END`
}
