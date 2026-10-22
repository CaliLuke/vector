// Package design is reverse-imported from the ticket collaboration source contract.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/ticket_collaboration/design --output ..
package design

import . "github.com/CaliLuke/loom/dsl"

var ImportedAccessEe6Enum = Type("AccessEe6Enum", String, func() {
	Meta("openapi:typename:canonical", "true")
	Description("* `INTERNAL` - INTERNAL\n* `EXTERNAL` - EXTERNAL")
	Enum("INTERNAL", "EXTERNAL")
})

var ImportedPatchedPlaneAPISerializersIssueIssuecommentcreateserializerRequestRequest = Type("PatchedPlaneApiSerializersIssueIssuecommentcreateserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for creating work item comments.\n\nHandles comment creation with JSON and HTML content support,\naccess control, and external integration tracking.")
	Attribute("access", ImportedAccessEe6Enum)
	Attribute("comment_html", String)
	Attribute("comment_json", Any, func() {
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
})

var ImportedPatchedPlaneAPISerializersIssueIssuelinkupdateserializerRequestRequest = Type("PatchedPlaneApiSerializersIssueIssuelinkupdateserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for updating work item external links.\n\nExtends link creation with update-specific validation to prevent\nURL conflicts and maintain link integrity during modifications.")
	Attribute("title", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("url", String, func() {
		MinLength(1)
	})
})

var ImportedPatchedPlaneAppSerializersIssueIssuecommentserializerRequestRequest = Type("PatchedPlaneAppSerializersIssueIssuecommentserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("access", ImportedAccessEe6Enum)
	Attribute("actor", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("attachments", ArrayOf(String, func() {
		Format(FormatURI)
		MinLength(1)
		MaxLength(200)
	}), func() {
		MaxLength(10)
	})
	Attribute("comment_html", String)
	Attribute("comment_json", Any, func() {
	})
	Attribute("comment_stripped", String, func() {
		Title("Comment")
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("description", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("edited_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("parent", String, func() {
		Nullable()
		Format(FormatUUID)
	})
})

var ImportedPatchedPlaneAppSerializersIssueIssuelinkserializerRequestRequest = Type("PatchedPlaneAppSerializersIssueIssuelinkserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("metadata", Any, func() {
	})
	Attribute("title", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("url", String, func() {
		MinLength(1)
	})
})

var ImportedPlaneAPISerializersIssueIssuecommentcreateserializerRequestRequest = Type("PlaneApiSerializersIssueIssuecommentcreateserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for creating work item comments.\n\nHandles comment creation with JSON and HTML content support,\naccess control, and external integration tracking.")
	Attribute("access", ImportedAccessEe6Enum)
	Attribute("comment_html", String)
	Attribute("comment_json", Any, func() {
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
})

var ImportedPlaneAPISerializersIssueIssuecommentserializerResponse = Type("PlaneApiSerializersIssueIssuecommentserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Full serializer for work item comments with membership context.\n\nProvides complete comment data including member status, content formatting,\nand edit tracking for collaborative work item discussions.")
	Attribute("access", ImportedAccessEe6Enum)
	Attribute("actor", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("attachments", ArrayOf(String, func() {
		Format(FormatURI)
		MaxLength(200)
	}), func() {
		MaxLength(10)
	})
	Attribute("comment_html", String)
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("description", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("edited_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_member", Boolean, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("issue", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("parent", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("updated_at", String, func() {
		Title("Last Modified At")
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
})

var ImportedPlaneAPISerializersIssueIssuelinkcreateserializerRequestRequest = Type("PlaneApiSerializersIssueIssuelinkcreateserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for creating work item external links with validation.\n\nHandles URL validation, format checking, and duplicate prevention\nfor attaching external resources to work items.")
	Attribute("title", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("url", String, func() {
		MinLength(1)
	})
	Required("url")
})

var ImportedPlaneAPISerializersIssueIssuelinkserializerResponse = Type("PlaneApiSerializersIssueIssuelinkserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Full serializer for work item external links.\n\nProvides complete link information including metadata and timestamps\nfor managing external resource associations with work items.")
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("issue", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("metadata", Any, func() {
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("title", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("updated_at", String, func() {
		Title("Last Modified At")
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("url", String)
	Attribute("workspace", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Required("url")
})

var ImportedPlaneAppSerializersIssueCommentreactionserializerRequestRequest = Type("PlaneAppSerializersIssueCommentreactionserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("reaction", String, func() {
		MinLength(1)
	})
	Required("reaction")
})

var ImportedPlaneAppSerializersIssueCommentreactionserializerResponse = Type("PlaneAppSerializersIssueCommentreactionserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("actor", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("comment", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("display_name", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("reaction", String)
	Attribute("updated_at", String, func() {
		Title("Last Modified At")
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Required("reaction")
})

var ImportedPlaneAppSerializersIssueIssuecommentserializerRequestRequest = Type("PlaneAppSerializersIssueIssuecommentserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("access", ImportedAccessEe6Enum)
	Attribute("actor", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("attachments", ArrayOf(String, func() {
		Format(FormatURI)
		MinLength(1)
		MaxLength(200)
	}), func() {
		MaxLength(10)
	})
	Attribute("comment_html", String)
	Attribute("comment_json", Any, func() {
	})
	Attribute("comment_stripped", String, func() {
		Title("Comment")
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("description", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("edited_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("parent", String, func() {
		Nullable()
		Format(FormatUUID)
	})
})

var ImportedPlaneAppSerializersIssueIssuecommentserializerResponse = Type("PlaneAppSerializersIssueIssuecommentserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("access", ImportedAccessEe6Enum)
	Attribute("actor", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("actor_detail", ImportedPlaneAppSerializersUserUserliteserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("attachments", ArrayOf(String, func() {
		Format(FormatURI)
		MaxLength(200)
	}), func() {
		MaxLength(10)
	})
	Attribute("comment_html", String)
	Attribute("comment_json", Any, func() {
	})
	Attribute("comment_reactions", ArrayOf(ImportedPlaneAppSerializersIssueCommentreactionserializerResponse), func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("comment_stripped", String, func() {
		Title("Comment")
	})
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("description", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("edited_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_member", Boolean, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("issue", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("issue_detail", ImportedPlaneAppSerializersIssueIssueflatserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("parent", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("project_detail", ImportedPlaneAppSerializersProjectProjectliteserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("updated_at", String, func() {
		Title("Last Modified At")
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("workspace_detail", ImportedPlaneAppSerializersWorkspaceWorkspaceliteserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
})

var ImportedPlaneAppSerializersIssueIssueflatserializerResponse = Type("PlaneAppSerializersIssueIssueflatserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("description_html", String)
	Attribute("description_json", Any, func() {
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_draft", Boolean)
	Attribute("name", String, func() {
		Title("Issue Name")
		MaxLength(255)
	})
	Attribute("priority", ImportedPriorityEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Title("Issue Priority")
	})
	Attribute("sequence_id", Int, func() {
		Title("Issue Sequence ID")
		Meta("openapi:format", "")
		Minimum(-2.147483648e+09)
		Maximum(2.147483647e+09)
	})
	Attribute("sort_order", Float64)
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Required("name")
})

var ImportedPlaneAppSerializersIssueIssuelinkserializerRequestRequest = Type("PlaneAppSerializersIssueIssuelinkserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("metadata", Any, func() {
	})
	Attribute("title", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("url", String, func() {
		MinLength(1)
	})
	Required("url")
})

var ImportedPlaneAppSerializersIssueIssuelinkserializerResponse = Type("PlaneAppSerializersIssueIssuelinkserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_by_detail", ImportedPlaneAppSerializersUserUserliteserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("issue", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("metadata", Any, func() {
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("title", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("updated_at", String, func() {
		Title("Last Modified At")
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("url", String)
	Attribute("workspace", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Required("url")
})

var ImportedPlaneAppSerializersIssueIssuereactionserializerRequestRequest = Type("PlaneAppSerializersIssueIssuereactionserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("reaction", String, func() {
		MinLength(1)
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
	})
	Required("reaction")
})

var ImportedPlaneAppSerializersIssueIssuereactionserializerResponse = Type("PlaneAppSerializersIssueIssuereactionserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("actor", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("actor_detail", ImportedPlaneAppSerializersUserUserliteserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("issue", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("reaction", String)
	Attribute("updated_at", String, func() {
		Title("Last Modified At")
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Required("reaction")
})

var ImportedPlaneAppSerializersIssueIssuesubscriberserializerRequestRequest = Type("PlaneAppSerializersIssueIssuesubscriberserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("subscriber", String, func() {
		Format(FormatUUID)
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
	})
	Required("subscriber")
})

var ImportedPlaneAppSerializersIssueIssuesubscriberserializerResponse = Type("PlaneAppSerializersIssueIssuesubscriberserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("issue", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("subscriber", String, func() {
		Format(FormatUUID)
	})
	Attribute("updated_at", String, func() {
		Title("Last Modified At")
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Required("subscriber")
})

var ImportedPlaneAppSerializersProjectProjectliteserializerResponse = Type("PlaneAppSerializersProjectProjectliteserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("cover_image", String, func() {
		Nullable()
		Meta("openapi:readOnly", "true")
	})
	Attribute("cover_image_url", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("description", String, func() {
		Title("Project Description")
		Meta("openapi:readOnly", "true")
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("identifier", String, func() {
		Title("Project Identifier")
		Meta("openapi:readOnly", "true")
	})
	Attribute("logo_props", Any, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("name", String, func() {
		Title("Project Name")
		Meta("openapi:readOnly", "true")
	})
})

var ImportedPlaneAppSerializersUserUserliteserializerResponse = Type("PlaneAppSerializersUserUserliteserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("avatar", String)
	Attribute("avatar_url", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("display_name", String, func() {
		MaxLength(255)
	})
	Attribute("first_name", String, func() {
		MaxLength(255)
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_bot", Boolean, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("last_name", String, func() {
		MaxLength(255)
	})
})

var ImportedPlaneAppSerializersWorkspaceWorkspaceliteserializerResponse = Type("PlaneAppSerializersWorkspaceWorkspaceliteserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("logo_url", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("name", String, func() {
		Title("Workspace Name")
		Meta("openapi:readOnly", "true")
	})
	Attribute("slug", String, func() {
		Meta("openapi:readOnly", "true")
		Pattern("^[-a-zA-Z0-9_]+$")
	})
})

var ImportedPriorityEnum = Type("PriorityEnum", String, func() {
	Meta("openapi:typename:canonical", "true")
	Description("* `urgent` - Urgent\n* `high` - High\n* `medium` - Medium\n* `low` - Low\n* `none` - None")
	Enum("urgent", "high", "medium", "low", "none")
})

var ImportedRestFrameworkSerializersPaginatedissuecommentresponseResponse = Type("RestFrameworkSerializersPaginatedissuecommentresponseResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("count", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("extra_stats", String, func() {
		Nullable()
	})
	Attribute("grouped_by", String, func() {
		Nullable()
	})
	Attribute("next_cursor", String)
	Attribute("next_page_results", Boolean)
	Attribute("prev_cursor", String)
	Attribute("prev_page_results", Boolean)
	Attribute("results", ArrayOf(ImportedPlaneAPISerializersIssueIssuecommentserializerResponse))
	Attribute("sub_grouped_by", String, func() {
		Nullable()
	})
	Attribute("total_count", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("total_pages", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("total_results", Int, func() {
		Meta("openapi:format", "")
	})
	Required("count", "extra_stats", "grouped_by", "next_cursor", "next_page_results", "prev_cursor", "prev_page_results", "results", "sub_grouped_by", "total_count", "total_pages", "total_results")
})

var ImportedRestFrameworkSerializersPaginatedissuelinkdetailresponseResponse = Type("RestFrameworkSerializersPaginatedissuelinkdetailresponseResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("count", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("extra_stats", String, func() {
		Nullable()
	})
	Attribute("grouped_by", String, func() {
		Nullable()
	})
	Attribute("next_cursor", String)
	Attribute("next_page_results", Boolean)
	Attribute("prev_cursor", String)
	Attribute("prev_page_results", Boolean)
	Attribute("results", ArrayOf(ImportedPlaneAPISerializersIssueIssuelinkserializerResponse))
	Attribute("sub_grouped_by", String, func() {
		Nullable()
	})
	Attribute("total_count", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("total_pages", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("total_results", Int, func() {
		Meta("openapi:format", "")
	})
	Required("count", "extra_stats", "grouped_by", "next_cursor", "next_page_results", "prev_cursor", "prev_page_results", "results", "sub_grouped_by", "total_count", "total_pages", "total_results")
})

var ImportedRestFrameworkSerializersPaginatedissuelinkresponseResponse = Type("RestFrameworkSerializersPaginatedissuelinkresponseResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("count", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("extra_stats", String, func() {
		Nullable()
	})
	Attribute("grouped_by", String, func() {
		Nullable()
	})
	Attribute("next_cursor", String)
	Attribute("next_page_results", Boolean)
	Attribute("prev_cursor", String)
	Attribute("prev_page_results", Boolean)
	Attribute("results", ArrayOf(ImportedPlaneAPISerializersIssueIssuelinkserializerResponse))
	Attribute("sub_grouped_by", String, func() {
		Nullable()
	})
	Attribute("total_count", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("total_pages", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("total_results", Int, func() {
		Meta("openapi:format", "")
	})
	Required("count", "extra_stats", "grouped_by", "next_cursor", "next_page_results", "prev_cursor", "prev_page_results", "results", "sub_grouped_by", "total_count", "total_pages", "total_results")
})

var ImportedAPIKeyAuthSecurity = APIKeySecurity("ApiKeyAuth", func() {
	Description("Plane API token authentication.")
})

var ImportedDjangoCsrfCookieSecurity = APIKeySecurity("DjangoCsrfCookie", func() {
	Description("Django CSRF cookie for unsafe standard-session requests.")
})

var ImportedDjangoCsrfHeaderSecurity = APIKeySecurity("DjangoCsrfHeader", func() {
	Description("Django CSRF header for unsafe standard-session requests.")
})

var ImportedDjangoSessionCookieSecurity = APIKeySecurity("DjangoSessionCookie", func() {
	Description("Standard Django session authentication.")
})

var ImportedPlaneAdminSessionCookieSecurity = APIKeySecurity("PlaneAdminSessionCookie", func() {
	Description("Plane instance-administration session authentication without CSRF enforcement.")
})

var ImportedPlaneSessionCookieSecurity = APIKeySecurity("PlaneSessionCookie", func() {
	Description("Plane session authentication without CSRF enforcement.")
})

var _ = API("Plane Internal Django Migration API", func() {
	Title("Plane Internal Django Migration API")
	Description("Internal full-server migration schema. Route inventory and behavior fixtures remain authoritative. This artifact does not change the public /api/v1 schema.")
	Version("1.0.0")
	Meta("openapi:example", "false")
	Meta("openapi:version", "3.1")
	Meta("openapi:tag:Assets")
	Meta("openapi:tag:Assets:desc", "**File Upload & Presigned URLs**\n\nGenerate presigned URLs for direct file uploads to cloud storage. Handle user avatars, cover images, and generic project assets with secure upload workflows.\n\n*Key Features:*\n- Generate presigned URLs for S3 uploads\n- Support for user avatars and cover images\n- Generic asset upload for projects\n- File validation and size limits\n\n*Use Cases:* User profile images, project file uploads, secure direct-to-cloud uploads.")
	Meta("openapi:tag:Cycles")
	Meta("openapi:tag:Cycles:desc", "**Sprint & Development Cycles**\n\nCreate and manage development cycles (sprints) to organize work into time-boxed iterations. Track progress, assign work items, and monitor team velocity.\n\n*Key Features:*\n- Create and configure development cycles\n- Assign work items to cycles\n- Track cycle progress and completion\n- Generate cycle analytics and reports\n\n*Use Cases:* Sprint planning, iterative development, progress tracking, team velocity.")
	Meta("openapi:tag:Intake")
	Meta("openapi:tag:Intake:desc", "**Work Item Intake Queue**\n\nManage incoming work items through a dedicated intake queue for triage and review. Submit, update, and process work items before they enter the main project workflow.\n\n*Key Features:*\n- Submit work items to intake queue\n- Review and triage incoming work items\n- Update intake work item status and properties\n- Accept, reject, or modify work items before approval\n\n*Use Cases:* Work item triage, external submissions, quality review, approval workflows.")
	Meta("openapi:tag:Labels")
	Meta("openapi:tag:Labels:desc", "**Labels & Tags**\n\nCreate and manage labels to categorize and organize work items. Use color-coded labels for easy identification, filtering, and project organization.\n\n*Key Features:*\n- Create custom labels with colors and descriptions\n- Apply labels to work items for categorization\n- Filter and search by labels\n- Organize labels across projects\n\n*Use Cases:* Priority marking, feature categorization, bug classification, team organization.")
	Meta("openapi:tag:Members")
	Meta("openapi:tag:Members:desc", "**Team Member Management**\n\nManage team members, roles, and permissions within projects and workspaces. Control access levels and track member participation.\n\n*Key Features:*\n- Invite and manage team members\n- Assign roles and permissions\n- Control project and workspace access\n- Track member activity and participation\n\n*Use Cases:* Team setup, access control, role management, collaboration.")
	Meta("openapi:tag:Modules")
	Meta("openapi:tag:Modules:desc", "**Feature Modules**\n\nGroup related work items into modules for better organization and tracking. Plan features, track progress, and manage deliverables at a higher level.\n\n*Key Features:*\n- Create and organize feature modules\n- Group work items by module\n- Track module progress and completion\n- Manage module leads and assignments\n\n*Use Cases:* Feature planning, release organization, progress tracking, team coordination.")
	Meta("openapi:tag:Projects")
	Meta("openapi:tag:Projects:desc", "**Project Management**\n\nCreate and manage projects to organize your development work. Configure project settings, manage team access, and control project visibility.\n\n*Key Features:*\n- Create, update, and delete projects\n- Configure project settings and preferences\n- Manage team access and permissions\n- Control project visibility and sharing\n\n*Use Cases:* Project setup, team collaboration, access control, project configuration.")
	Meta("openapi:tag:States")
	Meta("openapi:tag:States:desc", "**Workflow States**\n\nDefine custom workflow states for work items to match your team's process. Configure state transitions and track work item progress through different stages.\n\n*Key Features:*\n- Create custom workflow states\n- Configure state transitions and rules\n- Track work item progress through states\n- Set state-based permissions and automation\n\n*Use Cases:* Custom workflows, status tracking, process automation, progress monitoring.")
	Meta("openapi:tag:Users")
	Meta("openapi:tag:Users:desc", "**Current User Information**\n\nGet information about the currently authenticated user including profile details and account settings.\n\n*Key Features:*\n- Retrieve current user profile\n- Access user account information\n- View user preferences and settings\n- Get authentication context\n\n*Use Cases:* Profile display, user context, account information, authentication status.")
	Meta("openapi:tag:Work Item Activity")
	Meta("openapi:tag:Work Item Activity:desc", "**Activity History & Search**\n\nView activity history and search for work items across the workspace. Get detailed activity logs and find work items using text search.\n\n*Key Features:*\n- View work item activity history\n- Search work items across workspace\n- Track changes and modifications\n- Filter search results by project\n\n*Use Cases:* Activity tracking, work item discovery, change history, workspace search.")
	Meta("openapi:tag:Work Item Attachments")
	Meta("openapi:tag:Work Item Attachments:desc", "**Work Item File Attachments**\n\nGenerate presigned URLs for uploading files directly to specific work items. Upload and manage attachments associated with work items.\n\n*Key Features:*\n- Generate presigned URLs for work item attachments\n- Upload files directly to work items\n- Retrieve and manage attachment metadata\n- Delete attachments from work items\n\n*Use Cases:* Screenshots, error logs, design files, supporting documents.")
	Meta("openapi:tag:Work Item Comments")
	Meta("openapi:tag:Work Item Comments:desc", "**Comments & Discussions**\n\nAdd comments and discussions to work items for team collaboration. Support threaded conversations, mentions, and rich text formatting.\n\n*Key Features:*\n- Add comments to work items\n- Thread conversations and replies\n- Mention users and trigger notifications\n- Rich text and markdown support\n\n*Use Cases:* Team discussions, progress updates, code reviews, decision tracking.")
	Meta("openapi:tag:Work Item Links")
	Meta("openapi:tag:Work Item Links:desc", "**External Links & References**\n\nLink work items to external resources like documentation, repositories, or design files. Maintain connections between work items and external systems.\n\n*Key Features:*\n- Add external URL links to work items\n- Validate and preview linked resources\n- Organize links by type and category\n- Track link usage and access\n\n*Use Cases:* Documentation links, repository connections, design references, external tools.")
	Meta("openapi:tag:Work Items")
	Meta("openapi:tag:Work Items:desc", "**Work Items & Tasks**\n\nCreate and manage work items like tasks, bugs, features, and user stories. The core entities for tracking work in your projects.\n\n*Key Features:*\n- Create, update, and manage work items\n- Assign to team members and set priorities\n- Track progress through workflow states\n- Set due dates, estimates, and relationships\n\n*Use Cases:* Bug tracking, task management, feature development, sprint planning.")
	Meta("openapi:tag:work-item-comments")
	Meta("openapi:tag:work-items")
	Meta("openapi:tag:work-item-links")
	HTTP(func() {
		Consumes("application/json", "application/x-www-form-urlencoded", "multipart/form-data")
		Produces("application/json")
	})
})

// collaborationUUIDPath follows Django's URL converter, which accepts every
// canonical hexadecimal UUID shape without imposing RFC version or variant bits.
func collaborationUUIDPath() {
	Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
	Meta("openapi:format", "uuid")
}

var _ = Service("TicketCollaboration", func() {
	HTTP(func() { RuntimeCORS() })
	Method("GetAPIPublicAnchorByAnchorCommentsByCommentIDReactions", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_public_anchor_by_anchor_comments_by_comment_id_reactions")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/public/anchor/{anchor}/comments/{comment_id}/reactions/\"")
		Meta("openapi:extension:x-plane-handler", "\"list\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.space.views.issue.CommentReactionPublicViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("anchor", String)
			Attribute("comment_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("search", String, func() {
				Description("A search term.")
				Meta("openapi:allowEmptyValue", "false")
			})
			Required("anchor", "comment_id")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			GET("/api/public/anchor/{anchor}/comments/{comment_id}/reactions/")
			Param("anchor")
			Param("comment_id")
			Param("search")
			Response(200, func() {
				ContentType("application/json")
			})
		})
	})
	Method("PostAPIPublicAnchorByAnchorCommentsByCommentIDReactions", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_public_anchor_by_anchor_comments_by_comment_id_reactions")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/public/anchor/{anchor}/comments/{comment_id}/reactions/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.space.views.issue.CommentReactionPublicViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("anchor", String)
			Attribute("comment_id", String, func() {
				collaborationUUIDPath()
			})
			Required("anchor", "comment_id")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			POST("/api/public/anchor/{anchor}/comments/{comment_id}/reactions/")
			Param("anchor")
			Param("comment_id")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersIssueCommentreactionserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(201, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueCommentreactionserializerResponse)
			})
		})
	})
	Method("DeleteAPIPublicAnchorByAnchorCommentsByCommentIDReactionsByReactionCode", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_public_anchor_by_anchor_comments_by_comment_id_reactions_by_reaction_code")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/public/anchor/{anchor}/comments/{comment_id}/reactions/{reaction_code}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.space.views.issue.CommentReactionPublicViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("anchor", String)
			Attribute("comment_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("reaction_code", String)
			Required("anchor", "comment_id", "reaction_code")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			DELETE("/api/public/anchor/{anchor}/comments/{comment_id}/reactions/{reaction_code}/")
			Param("anchor")
			Param("comment_id")
			Param("reaction_code")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("GetAPIPublicAnchorByAnchorIssuesByIssueIDComments", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_public_anchor_by_anchor_issues_by_issue_id_comments")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/public/anchor/{anchor}/issues/{issue_id}/comments/\"")
		Meta("openapi:extension:x-plane-handler", "\"list\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.AllowAny\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.space.views.issue.IssueCommentPublicViewSet\"")
		SessionSecurity(PlaneSession)
		Security()
		Payload(func() {
			Attribute("anchor", String)
			Attribute("issueID2", String, func() {
				Meta("openapi:allowEmptyValue", "false")
				// Django validates filters after authorization and returns field errors.
				Meta("openapi:format", "uuid")
			})
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("search", String, func() {
				Description("A search term.")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("workspaceID", String, func() {
				Meta("openapi:allowEmptyValue", "false")
			})
			Required("anchor", "issue_id")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			GET("/api/public/anchor/{anchor}/issues/{issue_id}/comments/")
			Param("anchor")
			Param("issueID2:issue__id")
			Param("issue_id")
			Param("search")
			Param("workspaceID:workspace__id")
			Response(200, func() {
				ContentType("application/json")
			})
		})
	})
	Method("PostAPIPublicAnchorByAnchorIssuesByIssueIDComments", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_public_anchor_by_anchor_issues_by_issue_id_comments")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/public/anchor/{anchor}/issues/{issue_id}/comments/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.space.views.issue.IssueCommentPublicViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("anchor", String)
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Required("anchor", "issue_id")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			POST("/api/public/anchor/{anchor}/issues/{issue_id}/comments/")
			Param("anchor")
			Param("issue_id")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersIssueIssuecommentserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(201, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssuecommentserializerResponse)
			})
		})
	})
	Method("DeleteAPIPublicAnchorByAnchorIssuesByIssueIDCommentsByPk", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_public_anchor_by_anchor_issues_by_issue_id_comments_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/public/anchor/{anchor}/issues/{issue_id}/comments/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.space.views.issue.IssueCommentPublicViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("anchor", String)
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("pk", String, func() {
				collaborationUUIDPath()
			})
			Required("anchor", "issue_id", "pk")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			DELETE("/api/public/anchor/{anchor}/issues/{issue_id}/comments/{pk}/")
			Param("anchor")
			Param("issue_id")
			Param("pk")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("GetAPIPublicAnchorByAnchorIssuesByIssueIDCommentsByPk", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_public_anchor_by_anchor_issues_by_issue_id_comments_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/public/anchor/{anchor}/issues/{issue_id}/comments/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"retrieve\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.AllowAny\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.space.views.issue.IssueCommentPublicViewSet\"")
		SessionSecurity(PlaneSession)
		Security()
		Payload(func() {
			Attribute("anchor", String)
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("pk", String, func() {
				collaborationUUIDPath()
			})
			Required("anchor", "issue_id", "pk")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			GET("/api/public/anchor/{anchor}/issues/{issue_id}/comments/{pk}/")
			Param("anchor")
			Param("issue_id")
			Param("pk")
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssuecommentserializerResponse)
			})
		})
	})
	Method("PatchAPIPublicAnchorByAnchorIssuesByIssueIDCommentsByPk", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "patch_api_public_anchor_by_anchor_issues_by_issue_id_comments_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/public/anchor/{anchor}/issues/{issue_id}/comments/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"partial_update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.space.views.issue.IssueCommentPublicViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("anchor", String)
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("pk", String, func() {
				collaborationUUIDPath()
			})
			Required("anchor", "issue_id", "pk")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			PATCH("/api/public/anchor/{anchor}/issues/{issue_id}/comments/{pk}/")
			Param("anchor")
			Param("issue_id")
			Param("pk")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAppSerializersIssueIssuecommentserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssuecommentserializerResponse)
			})
		})
	})
	Method("GetAPIPublicAnchorByAnchorIssuesByIssueIDReactions", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_public_anchor_by_anchor_issues_by_issue_id_reactions")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/public/anchor/{anchor}/issues/{issue_id}/reactions/\"")
		Meta("openapi:extension:x-plane-handler", "\"list\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.space.views.issue.IssueReactionPublicViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("anchor", String)
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("search", String, func() {
				Description("A search term.")
				Meta("openapi:allowEmptyValue", "false")
			})
			Required("anchor", "issue_id")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			GET("/api/public/anchor/{anchor}/issues/{issue_id}/reactions/")
			Param("anchor")
			Param("issue_id")
			Param("search")
			Response(200, func() {
				ContentType("application/json")
			})
		})
	})
	Method("PostAPIPublicAnchorByAnchorIssuesByIssueIDReactions", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_public_anchor_by_anchor_issues_by_issue_id_reactions")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/public/anchor/{anchor}/issues/{issue_id}/reactions/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.space.views.issue.IssueReactionPublicViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("anchor", String)
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Required("anchor", "issue_id")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			POST("/api/public/anchor/{anchor}/issues/{issue_id}/reactions/")
			Param("anchor")
			Param("issue_id")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersIssueIssuereactionserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(201, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssuereactionserializerResponse)
			})
		})
	})
	Method("DeleteAPIPublicAnchorByAnchorIssuesByIssueIDReactionsByReactionCode", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_public_anchor_by_anchor_issues_by_issue_id_reactions_by_reaction_code")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/public/anchor/{anchor}/issues/{issue_id}/reactions/{reaction_code}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.space.views.issue.IssueReactionPublicViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("anchor", String)
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("reaction_code", String)
			Required("anchor", "issue_id", "reaction_code")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			DELETE("/api/public/anchor/{anchor}/issues/{issue_id}/reactions/{reaction_code}/")
			Param("anchor")
			Param("issue_id")
			Param("reaction_code")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDComments", func() {
		Description("Retrieve all comments for a work item.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/comments/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectLitePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueCommentListCreateAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("cursor", String, func() {
				Description("Pagination cursor for getting next set of results")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("expand", String, func() {
				Description("Comma-separated list of related fields to expand in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("fields", String, func() {
				Description("Comma-separated list of fields to include in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				collaborationUUIDPath()
			})
			Attribute("orderBy", String, func() {
				Description("Field to order results by. Prefix with '-' for descending order")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("perPage", String, func() {
				Description("Number of results per page (default: 20, max: 100)")
				Meta("openapi:allowEmptyValue", "false")
				Meta("openapi:format", "")
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				collaborationUUIDPath()
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()

		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/comments/")
			Param("cursor")
			Param("expand")
			Param("fields")
			Param("issue_id")
			Param("orderBy:order_by")
			Param("perPage:per_page")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("Paginated list of work item comments")
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersPaginatedissuecommentresponseResponse, func() {
					Meta("openapi:allOf:reference", "true")
					Example("PaginatedWorkItemComments", func() {
						Meta("openapi:example:summary", "Paginated Work Item Comments")
						Value(Val{"count": 20, "extra_stats": nil, "grouped_by": "state", "next_cursor": "20:1:0", "next_page_results": true, "prev_cursor": "20:0:0", "prev_page_results": false, "results": []any{Val{"created_at": "2024-01-15T12:00:00Z", "id": "550e8400-e29b-41d4-a716-446655440000", "name": "Sample Item"}}, "sub_grouped_by": "priority", "total_count": 150, "total_pages": 8, "total_results": 150})
					})
				})
			})
		})
	})
	Method("PostAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDComments", func() {
		Description("Add a new comment to a work item with HTML content.")
		Meta("openapi:operationId", "post_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/comments/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectLitePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueCommentListCreateAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				collaborationUUIDPath()
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			POST("/api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/comments/")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAPISerializersIssueIssuecommentcreateserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false, func() {
				Meta("openapi:allOf:reference", "true")
				Example("IssueCommentCreateSerializer", func() {
					Description("Example request for creating an issue comment")
					Value(Val{"comment_html": "<p>New comment content</p>", "external_id": "1234567890", "external_source": "github"})
				})
			})
			Response(201, func() {
				Description("Work item comment created successfully")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIssueIssuecommentserializerResponse)
			})
		})
	})
	Method("DeleteAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsByPk", func() {
		Description("Permanently remove a comment from a work item. Records deletion activity for audit purposes.")
		Meta("openapi:operationId", "delete_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/comments/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"delete\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectLitePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueCommentDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				collaborationUUIDPath()
			})
			Attribute("pk", String, func() {
				Description("Comment ID")
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				collaborationUUIDPath()
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "pk", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			DELETE("/api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/comments/{pk}/")
			Param("issue_id")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(204, func() {
				Description("Work item comment deleted successfully")
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsByPk", func() {
		Description("Retrieve details of a specific comment.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/comments/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectLitePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueCommentDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				collaborationUUIDPath()
			})
			Attribute("pk", String, func() {
				Description("Comment ID")
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				collaborationUUIDPath()
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "pk", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/comments/{pk}/")
			Param("issue_id")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("Work item comments")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIssueIssuecommentserializerResponse)
			})
		})
	})
	Method("PatchAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsByPk", func() {
		Description("Modify the content of an existing comment on a work item.")
		Meta("openapi:operationId", "patch_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/comments/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"patch\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectLitePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueCommentDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				collaborationUUIDPath()
			})
			Attribute("pk", String, func() {
				Description("Comment ID")
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				collaborationUUIDPath()
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "pk", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			PATCH("/api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/comments/{pk}/")
			Param("issue_id")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAPISerializersIssueIssuecommentcreateserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false, func() {
				Meta("openapi:allOf:reference", "true")
				Example("IssueCommentCreateSerializer", func() {
					Description("Example request for updating an issue comment")
					Value(Val{"comment_html": "<p>Updated comment content</p>", "external_id": "1234567890", "external_source": "github"})
				})
			})
			Response(200, func() {
				Description("Work item comment updated successfully")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIssueIssuecommentserializerResponse)
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDLinks", func() {
		Description("Retrieve all links associated with a work item. Supports filtering by URL, title, and metadata.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_links")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/links/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueLinkListCreateAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("cursor", String, func() {
				Description("Pagination cursor for getting next set of results")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("expand", String, func() {
				Description("Comma-separated list of related fields to expand in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("fields", String, func() {
				Description("Comma-separated list of fields to include in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				collaborationUUIDPath()
			})
			Attribute("orderBy", String, func() {
				Description("Field to order results by. Prefix with '-' for descending order")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("perPage", String, func() {
				Description("Number of results per page (default: 20, max: 100)")
				Meta("openapi:allowEmptyValue", "false")
				Meta("openapi:format", "")
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				collaborationUUIDPath()
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-links")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/links/")
			Param("cursor")
			Param("expand")
			Param("fields")
			Param("issue_id")
			Param("orderBy:order_by")
			Param("perPage:per_page")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("Paginated list of work item links")
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersPaginatedissuelinkresponseResponse)
			})
		})
	})
	Method("PostAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDLinks", func() {
		Description("Add a new external link to a work item with URL, title, and metadata.")
		Meta("openapi:operationId", "post_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_links")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/links/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueLinkListCreateAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				collaborationUUIDPath()
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-links")
			POST("/api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/links/")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAPISerializersIssueIssuelinkcreateserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true, func() {
				Meta("openapi:allOf:reference", "true")
				Example("IssueLinkCreateSerializer", func() {
					Description("Example request for creating an issue link")
					Value(Val{"title": "Example Link", "url": "https://example.com"})
				})
			})
			Response(201, func() {
				Description("Work item link created successfully")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIssueIssuelinkserializerResponse)
			})
		})
	})
	Method("DeleteAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDLinksByPk", func() {
		Description("Permanently remove an external link from a work item.")
		Meta("openapi:operationId", "delete_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_links_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/links/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"delete\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueLinkDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				collaborationUUIDPath()
			})
			Attribute("pk", String, func() {
				Description("Link ID")
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				collaborationUUIDPath()
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "pk", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-links")
			DELETE("/api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/links/{pk}/")
			Param("issue_id")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(204, func() {
				Description("Work item link deleted successfully")
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDLinksByPk", func() {
		Description("Retrieve details of a specific work item link.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_links_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/links/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueLinkDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("cursor", String, func() {
				Description("Pagination cursor for getting next set of results")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("expand", String, func() {
				Description("Comma-separated list of related fields to expand in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("fields", String, func() {
				Description("Comma-separated list of fields to include in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				collaborationUUIDPath()
			})
			Attribute("perPage", String, func() {
				Description("Number of results per page (default: 20, max: 100)")
				Meta("openapi:allowEmptyValue", "false")
				Meta("openapi:format", "")
			})
			Attribute("pk", String, func() {
				Description("Link ID")
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				collaborationUUIDPath()
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "pk", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-links")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/links/{pk}/")
			Param("cursor")
			Param("expand")
			Param("fields")
			Param("issue_id")
			Param("perPage:per_page")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("Work item link details or paginated list")
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersPaginatedissuelinkdetailresponseResponse)
			})
		})
	})
	Method("PatchAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDLinksByPk", func() {
		Description("Modify the URL, title, or metadata of an existing issue link.")
		Meta("openapi:operationId", "patch_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_links_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/links/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"patch\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueLinkDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				collaborationUUIDPath()
			})
			Attribute("pk", String, func() {
				Description("Link ID")
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				collaborationUUIDPath()
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "pk", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-links")
			PATCH("/api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/links/{pk}/")
			Param("issue_id")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAPISerializersIssueIssuelinkupdateserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false, func() {
				Meta("openapi:allOf:reference", "true")
				Example("IssueLinkUpdateSerializer", func() {
					Description("Example request for updating an issue link")
					Value(Val{"title": "Updated Link", "url": "https://example.com"})
				})
			})
			Response(200, func() {
				Description("Issue link updated successfully")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIssueIssuelinkserializerResponse)
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDComments", func() {
		Description("Retrieve all comments for a work item.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_comments")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/comments/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectLitePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueCommentListCreateAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("cursor", String, func() {
				Description("Pagination cursor for getting next set of results")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("expand", String, func() {
				Description("Comma-separated list of related fields to expand in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("fields", String, func() {
				Description("Comma-separated list of fields to include in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				collaborationUUIDPath()
			})
			Attribute("orderBy", String, func() {
				Description("Field to order results by. Prefix with '-' for descending order")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("perPage", String, func() {
				Description("Number of results per page (default: 20, max: 100)")
				Meta("openapi:allowEmptyValue", "false")
				Meta("openapi:format", "")
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				collaborationUUIDPath()
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()

		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/comments/")
			Param("cursor")
			Param("expand")
			Param("fields")
			Param("issue_id")
			Param("orderBy:order_by")
			Param("perPage:per_page")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("Paginated list of work item comments")
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersPaginatedissuecommentresponseResponse, func() {
					Meta("openapi:allOf:reference", "true")
					Example("PaginatedWorkItemComments", func() {
						Meta("openapi:example:summary", "Paginated Work Item Comments")
						Value(Val{"count": 20, "extra_stats": nil, "grouped_by": "state", "next_cursor": "20:1:0", "next_page_results": true, "prev_cursor": "20:0:0", "prev_page_results": false, "results": []any{Val{"created_at": "2024-01-15T12:00:00Z", "id": "550e8400-e29b-41d4-a716-446655440000", "name": "Sample Item"}}, "sub_grouped_by": "priority", "total_count": 150, "total_pages": 8, "total_results": 150})
					})
				})
			})
		})
	})
	Method("PostAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDComments", func() {
		Description("Add a new comment to a work item with HTML content.")
		Meta("openapi:operationId", "post_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_comments")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/comments/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectLitePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueCommentListCreateAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				collaborationUUIDPath()
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			POST("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/comments/")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAPISerializersIssueIssuecommentcreateserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false, func() {
				Meta("openapi:allOf:reference", "true")
				Example("IssueCommentCreateSerializer", func() {
					Description("Example request for creating an issue comment")
					Value(Val{"comment_html": "<p>New comment content</p>", "external_id": "1234567890", "external_source": "github"})
				})
			})
			Response(201, func() {
				Description("Work item comment created successfully")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIssueIssuecommentserializerResponse)
			})
		})
	})
	Method("DeleteAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDCommentsByPk", func() {
		Description("Permanently remove a comment from a work item. Records deletion activity for audit purposes.")
		Meta("openapi:operationId", "delete_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_comments_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/comments/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"delete\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectLitePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueCommentDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				collaborationUUIDPath()
			})
			Attribute("pk", String, func() {
				Description("Comment ID")
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				collaborationUUIDPath()
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "pk", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			DELETE("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/comments/{pk}/")
			Param("issue_id")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(204, func() {
				Description("Work item comment deleted successfully")
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDCommentsByPk", func() {
		Description("Retrieve details of a specific comment.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_comments_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/comments/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectLitePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueCommentDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				collaborationUUIDPath()
			})
			Attribute("pk", String, func() {
				Description("Comment ID")
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				collaborationUUIDPath()
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "pk", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/comments/{pk}/")
			Param("issue_id")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("Work item comments")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIssueIssuecommentserializerResponse)
			})
		})
	})
	Method("PatchAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDCommentsByPk", func() {
		Description("Modify the content of an existing comment on a work item.")
		Meta("openapi:operationId", "patch_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_comments_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/comments/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"patch\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectLitePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueCommentDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				collaborationUUIDPath()
			})
			Attribute("pk", String, func() {
				Description("Comment ID")
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				collaborationUUIDPath()
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "pk", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			PATCH("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/comments/{pk}/")
			Param("issue_id")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAPISerializersIssueIssuecommentcreateserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false, func() {
				Meta("openapi:allOf:reference", "true")
				Example("IssueCommentCreateSerializer", func() {
					Description("Example request for updating an issue comment")
					Value(Val{"comment_html": "<p>Updated comment content</p>", "external_id": "1234567890", "external_source": "github"})
				})
			})
			Response(200, func() {
				Description("Work item comment updated successfully")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIssueIssuecommentserializerResponse)
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDLinks", func() {
		Description("Retrieve all links associated with a work item. Supports filtering by URL, title, and metadata.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_links")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/links/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueLinkListCreateAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("cursor", String, func() {
				Description("Pagination cursor for getting next set of results")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("expand", String, func() {
				Description("Comma-separated list of related fields to expand in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("fields", String, func() {
				Description("Comma-separated list of fields to include in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				collaborationUUIDPath()
			})
			Attribute("orderBy", String, func() {
				Description("Field to order results by. Prefix with '-' for descending order")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("perPage", String, func() {
				Description("Number of results per page (default: 20, max: 100)")
				Meta("openapi:allowEmptyValue", "false")
				Meta("openapi:format", "")
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				collaborationUUIDPath()
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-links")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/links/")
			Param("cursor")
			Param("expand")
			Param("fields")
			Param("issue_id")
			Param("orderBy:order_by")
			Param("perPage:per_page")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("Paginated list of work item links")
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersPaginatedissuelinkresponseResponse)
			})
		})
	})
	Method("PostAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDLinks", func() {
		Description("Add a new external link to a work item with URL, title, and metadata.")
		Meta("openapi:operationId", "post_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_links")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/links/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueLinkListCreateAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				collaborationUUIDPath()
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-links")
			POST("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/links/")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAPISerializersIssueIssuelinkcreateserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true, func() {
				Meta("openapi:allOf:reference", "true")
				Example("IssueLinkCreateSerializer", func() {
					Description("Example request for creating an issue link")
					Value(Val{"title": "Example Link", "url": "https://example.com"})
				})
			})
			Response(201, func() {
				Description("Work item link created successfully")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIssueIssuelinkserializerResponse)
			})
		})
	})
	Method("DeleteAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDLinksByPk", func() {
		Description("Permanently remove an external link from a work item.")
		Meta("openapi:operationId", "delete_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_links_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/links/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"delete\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueLinkDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				collaborationUUIDPath()
			})
			Attribute("pk", String, func() {
				Description("Link ID")
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				collaborationUUIDPath()
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "pk", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-links")
			DELETE("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/links/{pk}/")
			Param("issue_id")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(204, func() {
				Description("Work item link deleted successfully")
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDLinksByPk", func() {
		Description("Retrieve details of a specific work item link.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_links_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/links/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueLinkDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("cursor", String, func() {
				Description("Pagination cursor for getting next set of results")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("expand", String, func() {
				Description("Comma-separated list of related fields to expand in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("fields", String, func() {
				Description("Comma-separated list of fields to include in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				collaborationUUIDPath()
			})
			Attribute("perPage", String, func() {
				Description("Number of results per page (default: 20, max: 100)")
				Meta("openapi:allowEmptyValue", "false")
				Meta("openapi:format", "")
			})
			Attribute("pk", String, func() {
				Description("Link ID")
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				collaborationUUIDPath()
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "pk", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-links")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/links/{pk}/")
			Param("cursor")
			Param("expand")
			Param("fields")
			Param("issue_id")
			Param("perPage:per_page")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("Work item link details or paginated list")
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersPaginatedissuelinkdetailresponseResponse)
			})
		})
	})
	Method("PatchAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDLinksByPk", func() {
		Description("Modify the URL, title, or metadata of an existing issue link.")
		Meta("openapi:operationId", "patch_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_links_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/links/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"patch\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueLinkDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				collaborationUUIDPath()
			})
			Attribute("pk", String, func() {
				Description("Link ID")
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				collaborationUUIDPath()
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "pk", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-links")
			PATCH("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/links/{pk}/")
			Param("issue_id")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAPISerializersIssueIssuelinkupdateserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false, func() {
				Meta("openapi:allOf:reference", "true")
				Example("IssueLinkUpdateSerializer", func() {
					Description("Example request for updating an issue link")
					Value(Val{"title": "Updated Link", "url": "https://example.com"})
				})
			})
			Response(200, func() {
				Description("Issue link updated successfully")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIssueIssuelinkserializerResponse)
			})
		})
	})
	Method("GetAPIWorkspacesBySlugProjectsByProjectIDCommentsByCommentIDReactions", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_comments_by_comment_id_reactions")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/projects/{project_id}/comments/{comment_id}/reactions/\"")
		Meta("openapi:extension:x-plane-handler", "\"list\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.comment.CommentReactionViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("comment_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("search", String, func() {
				Description("A search term.")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("slug", String)
			Required("comment_id", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			GET("/api/workspaces/{slug}/projects/{project_id}/comments/{comment_id}/reactions/")
			Param("comment_id")
			Param("project_id")
			Param("search")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDCommentsByCommentIDReactions", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_comments_by_comment_id_reactions")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/comments/{comment_id}/reactions/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.comment.CommentReactionViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("comment_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("slug", String)
			Required("comment_id", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			POST("/api/workspaces/{slug}/projects/{project_id}/comments/{comment_id}/reactions/")
			Param("comment_id")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersIssueCommentreactionserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(201, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueCommentreactionserializerResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDCommentsByCommentIDReactionsByReactionCode", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_comments_by_comment_id_reactions_by_reaction_code")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/comments/{comment_id}/reactions/{reaction_code}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.comment.CommentReactionViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("comment_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("reaction_code", String)
			Attribute("slug", String)
			Required("comment_id", "project_id", "reaction_code", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/comments/{comment_id}/reactions/{reaction_code}/")
			Param("comment_id")
			Param("project_id")
			Param("reaction_code")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDComments", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/comments/\"")
		Meta("openapi:extension:x-plane-handler", "\"list\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.comment.IssueCommentViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issueID2", String, func() {
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("search", String, func() {
				Description("A search term.")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("slug", String)
			Attribute("workspaceID", String, func() {
				Meta("openapi:allowEmptyValue", "false")
				// Django validates filters after authorization and returns field errors.
				Meta("openapi:format", "uuid")
			})
			Required("issue_id", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			GET("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/comments/")
			Param("issueID2:issue__id")
			Param("issue_id")
			Param("project_id")
			Param("search")
			Param("slug")
			Param("workspaceID:workspace__id")
			Response(200, func() {
				ContentType("application/json")
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDComments", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/comments/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.comment.IssueCommentViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("slug", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			POST("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/comments/")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersIssueIssuecommentserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(201, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssuecommentserializerResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsByPk", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/comments/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.comment.IssueCommentViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("pk", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("slug", String)
			Required("issue_id", "pk", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/comments/{pk}/")
			Param("issue_id")
			Param("pk")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsByPk", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/comments/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"retrieve\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.comment.IssueCommentViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("pk", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("slug", String)
			Required("issue_id", "pk", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			GET("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/comments/{pk}/")
			Param("issue_id")
			Param("pk")
			Param("project_id")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssuecommentserializerResponse)
			})
		})
	})
	Method("PatchAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsByPk", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "patch_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/comments/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"partial_update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.comment.IssueCommentViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("pk", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("slug", String)
			Required("issue_id", "pk", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			PATCH("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/comments/{pk}/")
			Param("issue_id")
			Param("pk")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAppSerializersIssueIssuecommentserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssuecommentserializerResponse)
			})
		})
	})
	Method("PutAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDCommentsByPk", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "put_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PUT /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/comments/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.comment.IssueCommentViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("pk", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("slug", String)
			Required("issue_id", "pk", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-item-comments")
			PUT("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/comments/{pk}/")
			Param("issue_id")
			Param("pk")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersIssueIssuecommentserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssuecommentserializerResponse)
			})
		})
	})
	Method("GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueLinks", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_issue_links")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-links/\"")
		Meta("openapi:extension:x-plane-handler", "\"list\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.link.IssueLinkViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("search", String, func() {
				Description("A search term.")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("slug", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			GET("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-links/")
			Param("issue_id")
			Param("project_id")
			Param("search")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueLinks", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_issue_links")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-links/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.link.IssueLinkViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("slug", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			POST("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-links/")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersIssueIssuelinkserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(201, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssuelinkserializerResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueLinksByPk", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_issue_links_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-links/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.link.IssueLinkViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("pk", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("slug", String)
			Required("issue_id", "pk", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-links/{pk}/")
			Param("issue_id")
			Param("pk")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueLinksByPk", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_issue_links_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-links/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"retrieve\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.link.IssueLinkViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("pk", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("slug", String)
			Required("issue_id", "pk", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			GET("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-links/{pk}/")
			Param("issue_id")
			Param("pk")
			Param("project_id")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssuelinkserializerResponse)
			})
		})
	})
	Method("PatchAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueLinksByPk", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "patch_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_issue_links_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-links/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"partial_update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.link.IssueLinkViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("pk", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("slug", String)
			Required("issue_id", "pk", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			PATCH("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-links/{pk}/")
			Param("issue_id")
			Param("pk")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAppSerializersIssueIssuelinkserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssuelinkserializerResponse)
			})
		})
	})
	Method("PutAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueLinksByPk", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "put_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_issue_links_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PUT /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-links/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.link.IssueLinkViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("pk", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("slug", String)
			Required("issue_id", "pk", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			PUT("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-links/{pk}/")
			Param("issue_id")
			Param("pk")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersIssueIssuelinkserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssuelinkserializerResponse)
			})
		})
	})
	Method("GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueSubscribers", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_issue_subscribers")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-subscribers/\"")
		Meta("openapi:extension:x-plane-handler", "\"list\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.subscriber.IssueSubscriberViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("search", String, func() {
				Description("A search term.")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("slug", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			GET("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-subscribers/")
			Param("issue_id")
			Param("project_id")
			Param("search")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueSubscribers", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_issue_subscribers")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-subscribers/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.subscriber.IssueSubscriberViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("slug", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			POST("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-subscribers/")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersIssueIssuesubscriberserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(201, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssuesubscriberserializerResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueSubscribersBySubscriberID", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_issue_subscribers_by_subscriber_id")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-subscribers/{subscriber_id}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.subscriber.IssueSubscriberViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("slug", String)
			Attribute("subscriber_id", String, func() {
				collaborationUUIDPath()
			})
			Required("issue_id", "project_id", "slug", "subscriber_id")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-subscribers/{subscriber_id}/")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			Param("subscriber_id")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDReactions", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_reactions")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/reactions/\"")
		Meta("openapi:extension:x-plane-handler", "\"list\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.reaction.IssueReactionViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("search", String, func() {
				Description("A search term.")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("slug", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			GET("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/reactions/")
			Param("issue_id")
			Param("project_id")
			Param("search")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDReactions", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_reactions")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/reactions/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.reaction.IssueReactionViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("slug", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			POST("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/reactions/")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersIssueIssuereactionserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(201, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssuereactionserializerResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDReactionsByReactionCode", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_reactions_by_reaction_code")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/reactions/{reaction_code}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.reaction.IssueReactionViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("reaction_code", String)
			Attribute("slug", String)
			Required("issue_id", "project_id", "reaction_code", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/reactions/{reaction_code}/")
			Param("issue_id")
			Param("project_id")
			Param("reaction_code")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDSubscribe", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_subscribe")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/subscribe/\"")
		Meta("openapi:extension:x-plane-handler", "\"unsubscribe\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectLitePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.subscriber.IssueSubscriberViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("slug", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/subscribe/")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDSubscribe", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_subscribe")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/subscribe/\"")
		Meta("openapi:extension:x-plane-handler", "\"subscription_status\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectLitePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.subscriber.IssueSubscriberViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("slug", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			GET("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/subscribe/")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssuesubscriberserializerResponse)
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDSubscribe", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_subscribe")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/subscribe/\"")
		Meta("openapi:extension:x-plane-handler", "\"subscribe\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectLitePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.subscriber.IssueSubscriberViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("project_id", String, func() {
				collaborationUUIDPath()
			})
			Attribute("slug", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(collaborationResponse)
		collaborationErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			POST("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/subscribe/")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersIssueIssuesubscriberserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(201, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssuesubscriberserializerResponse)
			})
		})
	})
})

// The source payloads accept JSON, form and multipart bodies. Their imported
// raw request streams preserve parser precedence and Django field coercions.
var PlaneSession = SessionAuth("PlaneSession", func() {
	CookieTransport(ImportedPlaneSessionCookieSecurity, "", func() { CookieName("session-id") })
})
var collaborationResponse = Type("CollaborationResponse", func() {})
var collaborationError = Type("CollaborationError", func() {
	ErrorName("kind", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Required("kind", "body")
})

func collaborationErrors() {
	for _, item := range []struct {
		name string
		code int
	}{
		{"status_400", 400}, {"status_401", 401}, {"status_403", 403},
		{"status_404", 404}, {"status_405", 405}, {"status_406", 406},
		{"status_409", 409}, {"status_415", 415}, {"status_429", 429}, {"status_500", 500},
	} {
		Error(item.name, collaborationError, func() {
			if item.code == 500 {
				Fault()
			}
			if item.code == 429 {
				Temporary()
			}
		})
		HTTP(func() {
			Response(item.name, item.code, func() { ContentType("application/json"); ErrorNameFromStatus(); Body("body") })
		})
	}
}
