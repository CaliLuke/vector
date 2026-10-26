// Package design is reverse-imported from the selected work-item source contract.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/work_item_core/design --output ..
package design

import . "github.com/CaliLuke/loom/dsl"

var bulkDeleteIssueResponse = Type("BulkDeleteIssueResponse", func() {
	Attribute("message", String)
	Required("message")
})

var ImportedGroupEnum = Type("GroupEnum", String, func() {
	Meta("openapi:typename:canonical", "true")
	Description("* `backlog` - Backlog\n* `unstarted` - Unstarted\n* `started` - Started\n* `completed` - Completed\n* `cancelled` - Cancelled\n* `triage` - Triage")
	Enum("backlog", "unstarted", "started", "completed", "cancelled", "triage")
})

var ImportedIntakeWorkItemStatusEnum = Type("IntakeWorkItemStatusEnum", Int, func() {
	Meta("openapi:typename:canonical", "true")
	Description("* `-2` - Pending\n* `-1` - Rejected\n* `0` - Snoozed\n* `1` - Accepted\n* `2` - Duplicate")
	Meta("openapi:format", "")
	Enum(-2, -1, 0, 1, 2)
})

var ImportedModuleStatusEnum = Type("ModuleStatusEnum", String, func() {
	Meta("openapi:typename:canonical", "true")
	Description("* `backlog` - Backlog\n* `planned` - Planned\n* `in-progress` - In Progress\n* `paused` - Paused\n* `completed` - Completed\n* `cancelled` - Cancelled")
	Enum("backlog", "planned", "in-progress", "paused", "completed", "cancelled")
})

var ImportedPatchedPlaneAPISerializersIntakeIntakeissueupdateserializerRequestRequest = Type("PatchedPlaneApiSerializersIntakeIntakeissueupdateserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for updating intake work items and their associated issues.\n\nHandles intake work item modifications including status changes, triage decisions,\nand embedded issue updates for issue queue processing workflows.")
	Attribute("duplicate_to", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("issue", ImportedPlaneAPISerializersIntakeIssueforintakeserializerRequestRequest, func() {
		Meta("openapi:allOf:reference", "true")
	})
	Attribute("snoozed_till", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("source_email", String, func() {
		Nullable()
	})
	Attribute("status", ImportedIntakeWorkItemStatusEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum(-2.147483648e+09)
		Maximum(2.147483647e+09)
	})
})

var ImportedPatchedPlaneAPISerializersIssueIssueserializerRequestRequest = Type("PatchedPlaneApiSerializersIssueIssueserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Comprehensive work item serializer with full relationship management.\n\nHandles complete work item lifecycle including assignees, labels, validation,\nand related model updates. Supports dynamic field expansion and HTML content\nprocessing.")
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("assignees", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:writeOnly", "true")
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("description_html", String)
	Attribute("estimate_point", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("is_draft", Boolean)
	Attribute("labels", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:writeOnly", "true")
	})
	Attribute("name", String, func() {
		Title("Issue Name")
		MinLength(1)
		MaxLength(255)
	})
	Attribute("parent", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("point", Int, func() {
		Nullable()
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(12)
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
	Attribute("state", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("type", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("type_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
})

var ImportedPatchedPlaneAppSerializersCycleCycleissueserializerRequestRequest = Type("PatchedPlaneAppSerializersCycleCycleissueserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("issue", String, func() {
		Format(FormatUUID)
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
	})
})

var ImportedPatchedPlaneAppSerializersDraftDraftissuecreateserializerRequestRequest = Type("PatchedPlaneAppSerializersDraftDraftissuecreateserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("assignee_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:writeOnly", "true")
	})
	Attribute("completed_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("description_html", String)
	Attribute("description_json", Any, func() {
	})
	Attribute("description_stripped", String, func() {
		Nullable()
	})
	Attribute("estimate_point", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("label_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:writeOnly", "true")
	})
	Attribute("name", String, func() {
		Nullable()
		Title("Issue Name")
		MaxLength(255)
	})
	Attribute("parent", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("parent_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("priority", ImportedPriorityEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Title("Issue Priority")
	})
	Attribute("project", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("sort_order", Float64)
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("state", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("state_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("type", String, func() {
		Nullable()
		Format(FormatUUID)
	})
})

var ImportedPatchedPlaneAppSerializersIntakeIntakeissueserializerRequestRequest = Type("PatchedPlaneAppSerializersIntakeIntakeissueserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("duplicate_to", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("snoozed_till", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("status", ImportedIntakeWorkItemStatusEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum(-2.147483648e+09)
		Maximum(2.147483647e+09)
	})
})

var ImportedPatchedPlaneAppSerializersIssueIssuecreateserializerRequestRequest = Type("PatchedPlaneAppSerializersIssueIssuecreateserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("assignee_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:writeOnly", "true")
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("description_html", String)
	Attribute("description_json", Any, func() {
	})
	Attribute("description_stripped", String, func() {
		Nullable()
	})
	Attribute("estimate_point", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("is_draft", Boolean)
	Attribute("label_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:writeOnly", "true")
	})
	Attribute("name", String, func() {
		Title("Issue Name")
		MinLength(1)
		MaxLength(255)
	})
	Attribute("parent", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("parent_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("point", Int, func() {
		Nullable()
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(12)
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
	Attribute("state", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("state_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("type", String, func() {
		Nullable()
		Format(FormatUUID)
	})
})

var ImportedPatchedPlaneAppSerializersModuleModuleissueserializerRequestRequest = Type("PatchedPlaneAppSerializersModuleModuleissueserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("issue", String, func() {
		Format(FormatUUID)
	})
})

var ImportedPlaneAPISerializersCycleCycleissuerequestserializerRequestRequest = Type("PlaneApiSerializersCycleCycleissuerequestserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for bulk work item assignment to cycles.\n\nValidates work item ID lists for batch operations including\ncycle assignment and sprint planning workflows.")
	Attribute("issues", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
	})
	Required("issues")
})

var ImportedPlaneAPISerializersCycleCycleissueserializerResponse = Type("PlaneApiSerializersCycleCycleissueserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for cycle-issue relationships with sub-issue counting.\n\nManages the association between cycles and work items, including\nhierarchical issue tracking for nested work item structures.")
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("cycle", String, func() {
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
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("sub_issues_count", Int, func() {
		Meta("openapi:format", "")
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
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Required("issue")
})

var ImportedPlaneAPISerializersCycleCycleliteserializerResponse = Type("PlaneApiSerializersCycleCycleliteserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Lightweight cycle serializer for minimal data transfer.\n\nProvides essential cycle information without computed metrics,\noptimized for list views and reference lookups.")
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDateTime)
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
	})
	Attribute("description", String, func() {
		Title("Cycle Description")
	})
	Attribute("end_date", String, func() {
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
	Attribute("logo_props", Any, func() {
	})
	Attribute("name", String, func() {
		Title("Cycle Name")
		MaxLength(255)
	})
	Attribute("owned_by", String, func() {
		Format(FormatUUID)
	})
	Attribute("progress_snapshot", Any, func() {
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
	})
	Attribute("sort_order", Float64)
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("timezone", ImportedTimezoneEnum)
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
	Attribute("version", Int, func() {
		Meta("openapi:format", "")
		Minimum(-2.147483648e+09)
		Maximum(2.147483647e+09)
	})
	Attribute("view_props", Any, func() {
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
	})
	Required("name", "owned_by", "project", "workspace")
})

var ImportedPlaneAPISerializersCycleTransfercycleissuerequestserializerRequestRequest = Type("PlaneApiSerializersCycleTransfercycleissuerequestserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for transferring work items between cycles.\n\nHandles work item migration between cycles including validation\nand relationship updates for sprint reallocation workflows.")
	Attribute("new_cycle_id", String, func() {
		Format(FormatUUID)
	})
	Required("new_cycle_id")
})

var ImportedPlaneAPISerializersIntakeIntakeissuecreateserializerRequestRequest = Type("PlaneApiSerializersIntakeIntakeissuecreateserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for creating intake work items with embedded issue data.\n\nManages intake work item creation including nested issue creation,\nstatus assignment, and source tracking for issue queue management.")
	Attribute("issue", ImportedPlaneAPISerializersIntakeIssueforintakeserializerRequestRequest, func() {
		Meta("openapi:allOf:reference", "true")
	})
	Required("issue")
})

var ImportedPlaneAPISerializersIntakeIntakeissueserializerResponse = Type("PlaneApiSerializersIntakeIntakeissueserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Comprehensive serializer for intake work items with expanded issue details.\n\nProvides full intake work item data including embedded issue information,\nstatus tracking, and triage metadata for issue queue management.")
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
	Attribute("duplicate_to", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("extra", Any, func() {
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("inbox", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("intake", String, func() {
		Format(FormatUUID)
	})
	Attribute("issue", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("issue_detail", ImportedPlaneAPISerializersIssueIssueexpandserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("snoozed_till", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("source_email", String, func() {
		Nullable()
	})
	Attribute("status", ImportedIntakeWorkItemStatusEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum(-2.147483648e+09)
		Maximum(2.147483647e+09)
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
	Required("intake")
})

var ImportedPlaneAPISerializersIntakeIssueforintakeserializerRequestRequest = Type("PlaneApiSerializersIntakeIssueforintakeserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for work item data within intake submissions.\n\nHandles essential work item fields for intake processing including\ncontent validation and priority assignment for triage workflows.")
	Attribute("description", Any, func() {
	})
	Attribute("description_html", String)
	Attribute("description_json", Any, func() {
	})
	Attribute("name", String, func() {
		Title("Issue Name")
		MinLength(1)
		MaxLength(255)
	})
	Attribute("priority", ImportedPriorityEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Title("Issue Priority")
	})
	Required("name")
})

var ImportedPlaneAPISerializersIssueIssueexpandserializerResponse = Type("PlaneApiSerializersIssueIssueexpandserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Extended work item serializer with full relationship expansion.\n\nProvides work items with expanded related data including cycles, modules,\nlabels, assignees, and states for comprehensive data representation.")
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("assignees", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("completed_at", String, func() {
		Nullable()
		Format(FormatDateTime)
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
	Attribute("cycle", ImportedPlaneAPISerializersCycleCycleliteserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("description", Any, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("description_binary", Bytes, func() {
		Nullable()
		Meta("openapi:format", "byte")
		Meta("openapi:readOnly", "true")
	})
	Attribute("description_html", String)
	Attribute("description_json", Any, func() {
	})
	Attribute("description_stripped", String, func() {
		Nullable()
	})
	Attribute("estimate_point", String, func() {
		Nullable()
		Format(FormatUUID)
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
	Attribute("is_draft", Boolean)
	Attribute("labels", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("module", ImportedPlaneAPISerializersModuleModuleliteserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("name", String, func() {
		Title("Issue Name")
		MaxLength(255)
	})
	Attribute("parent", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("point", Int, func() {
		Nullable()
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(12)
	})
	Attribute("priority", ImportedPriorityEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Title("Issue Priority")
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
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
	Attribute("state", ImportedPlaneAPISerializersStateStateliteserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("type", String, func() {
		Nullable()
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
		Meta("openapi:readOnly", "true")
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Required("name")
})

var ImportedPlaneAPISerializersIssueIssuerelationcreateserializerRequestRequest = Type("PlaneApiSerializersIssueIssuerelationcreateserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for creating issue relations.\n\nCreates issue relations with the specified relation type and issues.\nValidates relation types and ensures proper issue ID format.")
	Attribute("issues", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		MinLength(1)
	})
	Attribute("relation_type", ImportedRelationTypeEnum, func() {
		Meta("openapi:allOf:reference", "true")
	})
	Required("issues", "relation_type")
})

var ImportedPlaneAPISerializersIssueIssuerelationrefserializerResponse = Type("PlaneApiSerializersIssueIssuerelationrefserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Project-scoped reference to a related work item.")
	Attribute("issue_id", String, func() {
		Format(FormatUUID)
	})
	Attribute("project_id", String, func() {
		Format(FormatUUID)
	})
	Required("issue_id", "project_id")
})

var ImportedPlaneAPISerializersIssueIssuerelationresponseserializerResponse = Type("PlaneApiSerializersIssueIssuerelationresponseserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for issue relations response showing grouped relation types.\n\nEach list contains project_id and issue_id pairs so clients can resolve\ncross-project relations.")
	Attribute("blocked_by", ArrayOf(ImportedPlaneAPISerializersIssueIssuerelationrefserializerResponse), func() {
	})
	Attribute("blocking", ArrayOf(ImportedPlaneAPISerializersIssueIssuerelationrefserializerResponse), func() {
	})
	Attribute("duplicate", ArrayOf(ImportedPlaneAPISerializersIssueIssuerelationrefserializerResponse), func() {
	})
	Attribute("finish_after", ArrayOf(ImportedPlaneAPISerializersIssueIssuerelationrefserializerResponse), func() {
	})
	Attribute("finish_before", ArrayOf(ImportedPlaneAPISerializersIssueIssuerelationrefserializerResponse), func() {
	})
	Attribute("relates_to", ArrayOf(ImportedPlaneAPISerializersIssueIssuerelationrefserializerResponse), func() {
	})
	Attribute("start_after", ArrayOf(ImportedPlaneAPISerializersIssueIssuerelationrefserializerResponse), func() {
	})
	Attribute("start_before", ArrayOf(ImportedPlaneAPISerializersIssueIssuerelationrefserializerResponse), func() {
	})
	Required("blocked_by", "blocking", "duplicate", "finish_after", "finish_before", "relates_to", "start_after", "start_before")
})

var ImportedPlaneAPISerializersIssueIssuerelationserializerResponse = Type("PlaneApiSerializersIssueIssuerelationserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for issue relationships showing related issue details.\n\nProvides comprehensive information about related issues including\nproject context, sequence ID, and relationship type.")
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("name", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("priority", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("project_id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("relation_type", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("sequence_id", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("state_id", String, func() {
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
})

var ImportedPlaneAPISerializersIssueIssuesearchserializerResponse = Type("PlaneApiSerializersIssueIssuesearchserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for work item search result data formatting.\n\nProvides standardized search result structure including work item identifiers,\nproject context, and workspace information for search API responses.")
	Attribute("id", String, func() {
	})
	Attribute("name", String, func() {
	})
	Attribute("project__identifier", String, func() {
	})
	Attribute("project_id", String, func() {
	})
	Attribute("sequence_id", String, func() {
	})
	Attribute("workspace__slug", String, func() {
	})
	Required("id", "name", "project__identifier", "project_id", "sequence_id", "workspace__slug")
})

var ImportedPlaneAPISerializersIssueIssueserializerRequestRequest = Type("PlaneApiSerializersIssueIssueserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Comprehensive work item serializer with full relationship management.\n\nHandles complete work item lifecycle including assignees, labels, validation,\nand related model updates. Supports dynamic field expansion and HTML content\nprocessing.")
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("assignees", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:writeOnly", "true")
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("description_html", String)
	Attribute("estimate_point", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("is_draft", Boolean)
	Attribute("labels", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:writeOnly", "true")
	})
	Attribute("name", String, func() {
		Title("Issue Name")
		MinLength(1)
		MaxLength(255)
	})
	Attribute("parent", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("point", Int, func() {
		Nullable()
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(12)
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
	Attribute("state", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("type", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("type_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Required("name")
})

var ImportedPlaneAPISerializersIssueIssueserializerResponse = Type("PlaneApiSerializersIssueIssueserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Comprehensive work item serializer with full relationship management.\n\nHandles complete work item lifecycle including assignees, labels, validation,\nand related model updates. Supports dynamic field expansion and HTML content\nprocessing.")
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("completed_at", String, func() {
		Nullable()
		Format(FormatDateTime)
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
	})
	Attribute("description_binary", Bytes, func() {
		Nullable()
		Meta("openapi:format", "byte")
		Meta("openapi:readOnly", "true")
	})
	Attribute("description_html", String)
	Attribute("estimate_point", String, func() {
		Nullable()
		Format(FormatUUID)
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
	Attribute("is_draft", Boolean)
	Attribute("name", String, func() {
		Title("Issue Name")
		MaxLength(255)
	})
	Attribute("parent", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("point", Int, func() {
		Nullable()
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(12)
	})
	Attribute("priority", ImportedPriorityEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Title("Issue Priority")
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
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
	Attribute("state", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("type", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("type_id", String, func() {
		Nullable()
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
		Meta("openapi:readOnly", "true")
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Required("name")
})

var ImportedPlaneAPISerializersModuleModuleissuerequestserializerRequestRequest = Type("PlaneApiSerializersModuleModuleissuerequestserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for bulk work item assignment to modules.\n\nValidates work item ID lists for batch operations including\nmodule assignment and work item organization workflows.")
	Attribute("issues", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
	})
	Required("issues")
})

var ImportedPlaneAPISerializersModuleModuleissueserializerResponse = Type("PlaneApiSerializersModuleModuleissueserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for module-work item relationships with sub-item counting.\n\nManages the association between modules and work items, including\nhierarchical issue tracking for nested work item structures.")
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
	})
	Attribute("module", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("sub_issues_count", Int, func() {
		Meta("openapi:format", "")
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
	Required("issue")
})

var ImportedPlaneAPISerializersModuleModuleliteserializerResponse = Type("PlaneApiSerializersModuleModuleliteserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Lightweight module serializer for minimal data transfer.\n\nProvides essential module information without computed metrics,\noptimized for list views and reference lookups.")
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDateTime)
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
	})
	Attribute("description", String, func() {
		Title("Module Description")
	})
	Attribute("description_html", Any, func() {
	})
	Attribute("description_text", Any, func() {
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
	Attribute("lead", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("logo_props", Any, func() {
	})
	Attribute("members", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("name", String, func() {
		Title("Module Name")
		MaxLength(255)
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
	})
	Attribute("sort_order", Float64)
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("status", ImportedModuleStatusEnum)
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
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
	Attribute("view_props", Any, func() {
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
	})
	Required("name", "project", "workspace")
})

var ImportedPlaneAPISerializersStateStateliteserializerResponse = Type("PlaneApiSerializersStateStateliteserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Lightweight state serializer for minimal data transfer.\n\nProvides essential state information including visual properties\nand grouping data optimized for UI display and filtering.")
	Attribute("color", String, func() {
		Title("State Color")
		Meta("openapi:readOnly", "true")
	})
	Attribute("group", ImportedGroupEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("name", String, func() {
		Title("State Name")
		Meta("openapi:readOnly", "true")
	})
})

var ImportedPlaneAppSerializersCycleCycleissueserializerRequestRequest = Type("PlaneAppSerializersCycleCycleissueserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("issue", String, func() {
		Format(FormatUUID)
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
	})
	Required("issue")
})

var ImportedPlaneAppSerializersCycleCycleissueserializerResponse = Type("PlaneAppSerializersCycleCycleissueserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("cycle", String, func() {
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
	})
	Attribute("issue_detail", ImportedPlaneAppSerializersIssueIssuestateserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("sub_issues_count", Int, func() {
		Meta("openapi:format", "")
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
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Required("issue")
})

var ImportedPlaneAppSerializersDraftDraftissuecreateserializerRequestRequest = Type("PlaneAppSerializersDraftDraftissuecreateserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("assignee_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:writeOnly", "true")
	})
	Attribute("completed_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("description_html", String)
	Attribute("description_json", Any, func() {
	})
	Attribute("description_stripped", String, func() {
		Nullable()
	})
	Attribute("estimate_point", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("label_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:writeOnly", "true")
	})
	Attribute("name", String, func() {
		Nullable()
		Title("Issue Name")
		MaxLength(255)
	})
	Attribute("parent", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("parent_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("priority", ImportedPriorityEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Title("Issue Priority")
	})
	Attribute("project", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("sort_order", Float64)
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("state", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("state_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("type", String, func() {
		Nullable()
		Format(FormatUUID)
	})
})

var ImportedPlaneAppSerializersIntakeIntakeissueserializerRequestRequest = Type("PlaneAppSerializersIntakeIntakeissueserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("duplicate_to", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("snoozed_till", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("status", ImportedIntakeWorkItemStatusEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum(-2.147483648e+09)
		Maximum(2.147483647e+09)
	})
})

var ImportedPlaneAppSerializersIntakeIntakeissueserializerResponse = Type("PlaneAppSerializersIntakeIntakeissueserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("duplicate_to", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("issue", ImportedPlaneAppSerializersIssueIssueintakeserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("snoozed_till", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("status", ImportedIntakeWorkItemStatusEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum(-2.147483648e+09)
		Maximum(2.147483647e+09)
	})
})

var ImportedPlaneAppSerializersIssueIssuecreateserializerRequestRequest = Type("PlaneAppSerializersIssueIssuecreateserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("assignee_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:writeOnly", "true")
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("description_html", String)
	Attribute("description_json", Any, func() {
	})
	Attribute("description_stripped", String, func() {
		Nullable()
	})
	Attribute("estimate_point", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("is_draft", Boolean)
	Attribute("label_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:writeOnly", "true")
	})
	Attribute("name", String, func() {
		Title("Issue Name")
		MinLength(1)
		MaxLength(255)
	})
	Attribute("parent", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("parent_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("point", Int, func() {
		Nullable()
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(12)
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
	Attribute("state", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("state_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("type", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Required("name")
})

var ImportedPlaneAppSerializersIssueIssuecreateserializerResponse = Type("PlaneAppSerializersIssueIssuecreateserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("assignees", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("completed_at", String, func() {
		Nullable()
		Format(FormatDateTime)
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
	})
	Attribute("description_binary", Bytes, func() {
		Nullable()
		Meta("openapi:format", "byte")
		Meta("openapi:readOnly", "true")
	})
	Attribute("description_html", String)
	Attribute("description_json", Any, func() {
	})
	Attribute("description_stripped", String, func() {
		Nullable()
	})
	Attribute("estimate_point", String, func() {
		Nullable()
		Format(FormatUUID)
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
	Attribute("is_draft", Boolean)
	Attribute("labels", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("name", String, func() {
		Title("Issue Name")
		MaxLength(255)
	})
	Attribute("parent", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("parent_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("point", Int, func() {
		Nullable()
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(12)
	})
	Attribute("priority", ImportedPriorityEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Title("Issue Priority")
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("project_id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
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
	Attribute("state", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("state_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("type", String, func() {
		Nullable()
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
		Meta("openapi:readOnly", "true")
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("workspace_id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Required("name")
})

var ImportedPlaneAppSerializersIssueIssuedetailserializerResponse = Type("PlaneAppSerializersIssueIssuedetailserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDate)
		Meta("openapi:readOnly", "true")
	})
	Attribute("assignee_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("attachment_count", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("completed_at", String, func() {
		Nullable()
		Format(FormatDateTime)
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
	Attribute("cycle_id", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("description_html", String)
	Attribute("estimate_point", String, func() {
		Nullable()
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_draft", Boolean, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_intake", Boolean, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_subscribed", Boolean, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("label_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("link_count", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("module_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("name", String, func() {
		Title("Issue Name")
		Meta("openapi:readOnly", "true")
	})
	Attribute("parent_id", String, func() {
		Nullable()
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("priority", ImportedPriorityEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Title("Issue Priority")
		Meta("openapi:readOnly", "true")
	})
	Attribute("project_id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("sequence_id", Int, func() {
		Title("Issue Sequence ID")
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("sort_order", Float64, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDate)
		Meta("openapi:readOnly", "true")
	})
	Attribute("state_id", String, func() {
		Nullable()
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("sub_issues_count", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
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
	Required("description_html")
})

var ImportedPlaneAppSerializersIssueIssueflatserializerRequestRequest = Type("PlaneAppSerializersIssueIssueflatserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("description_html", String)
	Attribute("description_json", Any, func() {
	})
	Attribute("is_draft", Boolean)
	Attribute("name", String, func() {
		Title("Issue Name")
		MinLength(1)
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

var ImportedPlaneAppSerializersIssueIssueintakeserializerResponse = Type("PlaneAppSerializersIssueIssueintakeserializerResponse", func() {
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
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("label_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("name", String, func() {
		Title("Issue Name")
		Meta("openapi:readOnly", "true")
	})
	Attribute("priority", ImportedPriorityEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Title("Issue Priority")
		Meta("openapi:readOnly", "true")
	})
	Attribute("project_id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("sequence_id", Int, func() {
		Title("Issue Sequence ID")
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
})

var ImportedPlaneAppSerializersIssueIssuerelationserializerRequestRequest = Type("PlaneAppSerializersIssueIssuerelationserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("assignee_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:writeOnly", "true")
	})
})

var ImportedPlaneAppSerializersIssueIssuerelationserializerResponse = Type("PlaneAppSerializersIssueIssuerelationserializerResponse", func() {
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
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("name", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("priority", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("project_id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("relation_type", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("sequence_id", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("state_id", String, func() {
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
})

var ImportedPlaneAppSerializersIssueIssueserializerResponse = Type("PlaneAppSerializersIssueIssueserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDate)
		Meta("openapi:readOnly", "true")
	})
	Attribute("assignee_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("attachment_count", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("completed_at", String, func() {
		Nullable()
		Format(FormatDateTime)
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
	Attribute("cycle_id", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("estimate_point", String, func() {
		Nullable()
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_draft", Boolean, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("label_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("link_count", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("module_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("name", String, func() {
		Title("Issue Name")
		Meta("openapi:readOnly", "true")
	})
	Attribute("parent_id", String, func() {
		Nullable()
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("priority", ImportedPriorityEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Title("Issue Priority")
		Meta("openapi:readOnly", "true")
	})
	Attribute("project_id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("sequence_id", Int, func() {
		Title("Issue Sequence ID")
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("sort_order", Float64, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDate)
		Meta("openapi:readOnly", "true")
	})
	Attribute("state_id", String, func() {
		Nullable()
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("sub_issues_count", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
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
})

var ImportedPlaneAppSerializersIssueIssuestateserializerResponse = Type("PlaneAppSerializersIssueIssuestateserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("assignee_details", ArrayOf(ImportedPlaneAppSerializersUserUserliteserializerResponse), func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("assignees", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("attachment_count", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("completed_at", String, func() {
		Nullable()
		Format(FormatDateTime)
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
	})
	Attribute("description_binary", Bytes, func() {
		Nullable()
		Meta("openapi:format", "byte")
		Meta("openapi:readOnly", "true")
	})
	Attribute("description_html", String)
	Attribute("description_json", Any, func() {
	})
	Attribute("description_stripped", String, func() {
		Nullable()
	})
	Attribute("estimate_point", String, func() {
		Nullable()
		Format(FormatUUID)
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
	Attribute("is_draft", Boolean)
	Attribute("label_details", ArrayOf(ImportedPlaneAppSerializersIssueLabelliteserializerResponse), func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("labels", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("link_count", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("name", String, func() {
		Title("Issue Name")
		MaxLength(255)
	})
	Attribute("parent", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("point", Int, func() {
		Nullable()
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(12)
	})
	Attribute("priority", ImportedPriorityEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Title("Issue Priority")
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
	})
	Attribute("project_detail", ImportedPlaneAppSerializersProjectProjectliteserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
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
	Attribute("state", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("state_detail", ImportedPlaneAppSerializersStateStateliteserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("sub_issues_count", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("type", String, func() {
		Nullable()
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
	})
	Required("name", "project", "workspace")
})

var ImportedPlaneAppSerializersIssueLabelliteserializerResponse = Type("PlaneAppSerializersIssueLabelliteserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("color", String, func() {
		MaxLength(255)
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("name", String, func() {
		MaxLength(255)
	})
	Required("name")
})

var ImportedPlaneAppSerializersModuleModuleflatserializerResponse = Type("PlaneAppSerializersModuleModuleflatserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDateTime)
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
		Title("Module Description")
	})
	Attribute("description_html", Any, func() {
	})
	Attribute("description_text", Any, func() {
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
	Attribute("lead", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("logo_props", Any, func() {
	})
	Attribute("members", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("name", String, func() {
		Title("Module Name")
		MaxLength(255)
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("sort_order", Float64)
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("status", ImportedModuleStatusEnum)
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
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
	Attribute("view_props", Any, func() {
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Required("name")
})

var ImportedPlaneAppSerializersModuleModuleissueserializerRequestRequest = Type("PlaneAppSerializersModuleModuleissueserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("issue", String, func() {
		Format(FormatUUID)
	})
	Required("issue")
})

var ImportedPlaneAppSerializersModuleModuleissueserializerResponse = Type("PlaneAppSerializersModuleModuleissueserializerResponse", func() {
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
	})
	Attribute("issue_detail", ImportedPlaneAppSerializersProjectProjectliteserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("module", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("module_detail", ImportedPlaneAppSerializersModuleModuleflatserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("sub_issues_count", Int, func() {
		Meta("openapi:format", "")
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
	Required("issue")
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

var ImportedPlaneAppSerializersStateStateliteserializerResponse = Type("PlaneAppSerializersStateStateliteserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("color", String, func() {
		Title("State Color")
		Meta("openapi:readOnly", "true")
	})
	Attribute("group", ImportedGroupEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("name", String, func() {
		Title("State Name")
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

var ImportedPriorityEnum = Type("PriorityEnum", String, func() {
	Meta("openapi:typename:canonical", "true")
	Description("* `urgent` - Urgent\n* `high` - High\n* `medium` - Medium\n* `low` - Low\n* `none` - None")
	Enum("urgent", "high", "medium", "low", "none")
})

var ImportedRelationTypeEnum = Type("RelationTypeEnum", String, func() {
	Meta("openapi:typename:canonical", "true")
	Description("* `blocking` - Blocking\n* `blocked_by` - Blocked By\n* `duplicate` - Duplicate\n* `relates_to` - Relates To\n* `start_before` - Start Before\n* `start_after` - Start After\n* `finish_before` - Finish Before\n* `finish_after` - Finish After")
	Enum("blocking", "blocked_by", "duplicate", "relates_to", "start_before", "start_after", "finish_before", "finish_after")
})

var ImportedRestFrameworkSerializersArchiveresponseResponse = Type("RestFrameworkSerializersArchiveresponseResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("archived_at", String)
	Required("archived_at")
})

var ImportedRestFrameworkSerializersDeletedissueidsResponse = Type("RestFrameworkSerializersDeletedissueidsResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("id", String, func() {
		Format(FormatUUID)
	})
	Required("id")
})

var ImportedRestFrameworkSerializersDraftissuecreateresponseResponse = Type("RestFrameworkSerializersDraftissuecreateresponseResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("assignee_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("completed_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("cycle_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("description_html", String)
	Attribute("estimate_point", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
	})
	Attribute("label_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("module_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("name", String, func() {
		Nullable()
	})
	Attribute("parent_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("priority", String)
	Attribute("project_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("sort_order", Float64)
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("state_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("type_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("updated_at", String, func() {
		Format(FormatDateTime)
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Required("assignee_ids", "completed_at", "created_at", "created_by", "cycle_id", "description_html", "estimate_point", "id", "label_ids", "module_ids", "name", "parent_id", "priority", "project_id", "sort_order", "start_date", "state_id", "target_date", "type_id", "updated_at", "updated_by")
})

var ImportedRestFrameworkSerializersIssuedateupdateRequestRequest = Type("RestFrameworkSerializersIssuedateupdateRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("id", String, func() {
		Format(FormatUUID)
	})
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Required("id")
})

var ImportedRestFrameworkSerializersIssuedateupdatesrequestRequestRequest = Type("RestFrameworkSerializersIssuedateupdatesrequestRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("updates", ArrayOf(ImportedRestFrameworkSerializersIssuedateupdateRequestRequest))
	Required("updates")
})

var ImportedRestFrameworkSerializersIssueglobalpageResponse = Type("RestFrameworkSerializersIssueglobalpageResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("cursor", String)
	Attribute("next_cursor", String, func() {
		Nullable()
	})
	Attribute("next_page_results", Boolean)
	Attribute("page_count", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("prev_cursor", String, func() {
		Nullable()
	})
	Attribute("prev_page_results", Boolean)
	Attribute("results", ArrayOf(ImportedRestFrameworkSerializersIssueglobalpageitemResponse))
	Attribute("total_pages", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("total_results", Int, func() {
		Meta("openapi:format", "")
	})
	Required("cursor", "next_cursor", "next_page_results", "page_count", "prev_cursor", "prev_page_results", "results", "total_pages", "total_results")
})

var ImportedRestFrameworkSerializersIssueglobalpageitemResponse = Type("RestFrameworkSerializersIssueglobalpageitemResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("assignee_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("attachment_count", Int, func() {
		Nullable()
		Meta("openapi:format", "")
	})
	Attribute("completed_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("cycle_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("description_html", String)
	Attribute("estimate_point", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
	})
	Attribute("is_draft", Boolean)
	Attribute("label_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("link_count", Int, func() {
		Nullable()
		Meta("openapi:format", "")
	})
	Attribute("module_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("name", String)
	Attribute("parent_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("priority", String)
	Attribute("project_id", String, func() {
		Format(FormatUUID)
	})
	Attribute("sequence_id", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("sort_order", Float64)
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("state__group", String, func() {
		Nullable()
	})
	Attribute("state_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("sub_issues_count", Int, func() {
		Nullable()
		Meta("openapi:format", "")
	})
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("updated_at", String, func() {
		Format(FormatDateTime)
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Required("archived_at", "assignee_ids", "attachment_count", "completed_at", "created_at", "created_by", "cycle_id", "estimate_point", "id", "is_draft", "label_ids", "link_count", "module_ids", "name", "parent_id", "priority", "project_id", "sequence_id", "sort_order", "start_date", "state__group", "state_id", "sub_issues_count", "target_date", "updated_at", "updated_by")
})

var ImportedRestFrameworkSerializersIssueidsrequestRequestRequest = Type("RestFrameworkSerializersIssueidsrequestRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("issue_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Required("issue_ids")
})

var ImportedRestFrameworkSerializersIssuemodulesrequestRequestRequest = Type("RestFrameworkSerializersIssuemodulesrequestRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("modules", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("removed_modules", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
})

var ImportedRestFrameworkSerializersMessageresponseResponse = Type("RestFrameworkSerializersMessageresponseResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("message", String)
	Required("message")
})

var ImportedRestFrameworkSerializersModuleissuesrequestRequestRequest = Type("RestFrameworkSerializersModuleissuesrequestRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("issues", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Required("issues")
})

var ImportedRestFrameworkSerializersPaginatedintakeissueresponseResponse = Type("RestFrameworkSerializersPaginatedintakeissueresponseResponse", func() {
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
	Attribute("results", ArrayOf(ImportedPlaneAPISerializersIntakeIntakeissueserializerResponse))
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

var ImportedRestFrameworkSerializersPaginatedworkitemresponseResponse = Type("RestFrameworkSerializersPaginatedworkitemresponseResponse", func() {
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
	Attribute("results", ArrayOf(ImportedPlaneAPISerializersIssueIssueserializerResponse))
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

var ImportedRestFrameworkSerializersSubissueidsrequestRequestRequest = Type("RestFrameworkSerializersSubissueidsrequestRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("sub_issue_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Required("sub_issue_ids")
})

var ImportedRestFrameworkSerializersSubissuesresponseResponse = Type("RestFrameworkSerializersSubissuesresponseResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("state_distribution", MapOf(String, Any))
	Attribute("sub_issues", ArrayOf(ImportedPlaneAppSerializersIssueIssueserializerResponse))
	Required("state_distribution", "sub_issues")
})

var ImportedRestFrameworkSerializersTransfercycleissuesrequestRequestRequest = Type("RestFrameworkSerializersTransfercycleissuesrequestRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("new_cycle_id", String, func() {
		Format(FormatUUID)
	})
	Required("new_cycle_id")
})

var ImportedRestFrameworkSerializersViewissuelistitemResponse = Type("RestFrameworkSerializersViewissuelistitemResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("assignee_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("attachment_count", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("completed_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("cycle_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("estimate_point", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
	})
	Attribute("is_draft", Boolean)
	Attribute("label_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("link_count", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("module_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("name", String)
	Attribute("parent_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("priority", String)
	Attribute("project_id", String, func() {
		Format(FormatUUID)
	})
	Attribute("sequence_id", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("sort_order", Float64)
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("state__group", String, func() {
		Nullable()
	})
	Attribute("state_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("sub_issues_count", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("updated_at", String, func() {
		Format(FormatDateTime)
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Required("archived_at", "assignee_ids", "attachment_count", "completed_at", "created_at", "created_by", "cycle_id", "estimate_point", "id", "is_draft", "label_ids", "link_count", "module_ids", "name", "parent_id", "priority", "project_id", "sequence_id", "sort_order", "start_date", "state__group", "state_id", "sub_issues_count", "target_date", "updated_at", "updated_by")
})

var ImportedRestFrameworkSerializersWorkspaceuserissuespageResponse = Type("RestFrameworkSerializersWorkspaceuserissuespageResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("count", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("extra_stats", MapOf(String, Any), func() {
		Nullable()
	})
	Attribute("grouped_by", String, func() {
		Nullable()
	})
	Attribute("next_cursor", String)
	Attribute("next_page_results", Boolean)
	Attribute("prev_cursor", String)
	Attribute("prev_page_results", Boolean)
	Attribute("results", ArrayOf(ImportedRestFrameworkSerializersViewissuelistitemResponse))
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

var ImportedRestFrameworkSerializersWorkspaceviewissuespageResponse = Type("RestFrameworkSerializersWorkspaceviewissuespageResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("count", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("extra_stats", MapOf(String, Any), func() {
		Nullable()
	})
	Attribute("grouped_by", String, func() {
		Nullable()
	})
	Attribute("next_cursor", String)
	Attribute("next_page_results", Boolean)
	Attribute("prev_cursor", String)
	Attribute("prev_page_results", Boolean)
	Attribute("results", ArrayOf(ImportedRestFrameworkSerializersViewissuelistitemResponse))
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

var ImportedTimezoneEnum = Type("TimezoneEnum", String, func() {
	Meta("openapi:typename:canonical", "true")
	Description("* `Africa/Abidjan` - Africa/Abidjan\n* `Africa/Accra` - Africa/Accra\n* `Africa/Addis_Ababa` - Africa/Addis_Ababa\n* `Africa/Algiers` - Africa/Algiers\n* `Africa/Asmara` - Africa/Asmara\n* `Africa/Bamako` - Africa/Bamako\n* `Africa/Bangui` - Africa/Bangui\n* `Africa/Banjul` - Africa/Banjul\n* `Africa/Bissau` - Africa/Bissau\n* `Africa/Blantyre` - Africa/Blantyre\n* `Africa/Brazzaville` - Africa/Brazzaville\n* `Africa/Bujumbura` - Africa/Bujumbura\n* `Africa/Cairo` - Africa/Cairo\n* `Africa/Casablanca` - Africa/Casablanca\n* `Africa/Ceuta` - Africa/Ceuta\n* `Africa/Conakry` - Africa/Conakry\n* `Africa/Dakar` - Africa/Dakar\n* `Africa/Dar_es_Salaam` - Africa/Dar_es_Salaam\n* `Africa/Djibouti` - Africa/Djibouti\n* `Africa/Douala` - Africa/Douala\n* `Africa/El_Aaiun` - Africa/El_Aaiun\n* `Africa/Freetown` - Africa/Freetown\n* `Africa/Gaborone` - Africa/Gaborone\n* `Africa/Harare` - Africa/Harare\n* `Africa/Johannesburg` - Africa/Johannesburg\n* `Africa/Juba` - Africa/Juba\n* `Africa/Kampala` - Africa/Kampala\n* `Africa/Khartoum` - Africa/Khartoum\n* `Africa/Kigali` - Africa/Kigali\n* `Africa/Kinshasa` - Africa/Kinshasa\n* `Africa/Lagos` - Africa/Lagos\n* `Africa/Libreville` - Africa/Libreville\n* `Africa/Lome` - Africa/Lome\n* `Africa/Luanda` - Africa/Luanda\n* `Africa/Lubumbashi` - Africa/Lubumbashi\n* `Africa/Lusaka` - Africa/Lusaka\n* `Africa/Malabo` - Africa/Malabo\n* `Africa/Maputo` - Africa/Maputo\n* `Africa/Maseru` - Africa/Maseru\n* `Africa/Mbabane` - Africa/Mbabane\n* `Africa/Mogadishu` - Africa/Mogadishu\n* `Africa/Monrovia` - Africa/Monrovia\n* `Africa/Nairobi` - Africa/Nairobi\n* `Africa/Ndjamena` - Africa/Ndjamena\n* `Africa/Niamey` - Africa/Niamey\n* `Africa/Nouakchott` - Africa/Nouakchott\n* `Africa/Ouagadougou` - Africa/Ouagadougou\n* `Africa/Porto-Novo` - Africa/Porto-Novo\n* `Africa/Sao_Tome` - Africa/Sao_Tome\n* `Africa/Tripoli` - Africa/Tripoli\n* `Africa/Tunis` - Africa/Tunis\n* `Africa/Windhoek` - Africa/Windhoek\n* `America/Adak` - America/Adak\n* `America/Anchorage` - America/Anchorage\n* `America/Anguilla` - America/Anguilla\n* `America/Antigua` - America/Antigua\n* `America/Araguaina` - America/Araguaina\n* `America/Argentina/Buenos_Aires` - America/Argentina/Buenos_Aires\n* `America/Argentina/Catamarca` - America/Argentina/Catamarca\n* `America/Argentina/Cordoba` - America/Argentina/Cordoba\n* `America/Argentina/Jujuy` - America/Argentina/Jujuy\n* `America/Argentina/La_Rioja` - America/Argentina/La_Rioja\n* `America/Argentina/Mendoza` - America/Argentina/Mendoza\n* `America/Argentina/Rio_Gallegos` - America/Argentina/Rio_Gallegos\n* `America/Argentina/Salta` - America/Argentina/Salta\n* `America/Argentina/San_Juan` - America/Argentina/San_Juan\n* `America/Argentina/San_Luis` - America/Argentina/San_Luis\n* `America/Argentina/Tucuman` - America/Argentina/Tucuman\n* `America/Argentina/Ushuaia` - America/Argentina/Ushuaia\n* `America/Aruba` - America/Aruba\n* `America/Asuncion` - America/Asuncion\n* `America/Atikokan` - America/Atikokan\n* `America/Bahia` - America/Bahia\n* `America/Bahia_Banderas` - America/Bahia_Banderas\n* `America/Barbados` - America/Barbados\n* `America/Belem` - America/Belem\n* `America/Belize` - America/Belize\n* `America/Blanc-Sablon` - America/Blanc-Sablon\n* `America/Boa_Vista` - America/Boa_Vista\n* `America/Bogota` - America/Bogota\n* `America/Boise` - America/Boise\n* `America/Cambridge_Bay` - America/Cambridge_Bay\n* `America/Campo_Grande` - America/Campo_Grande\n* `America/Cancun` - America/Cancun\n* `America/Caracas` - America/Caracas\n* `America/Cayenne` - America/Cayenne\n* `America/Cayman` - America/Cayman\n* `America/Chicago` - America/Chicago\n* `America/Chihuahua` - America/Chihuahua\n* `America/Ciudad_Juarez` - America/Ciudad_Juarez\n* `America/Costa_Rica` - America/Costa_Rica\n* `America/Creston` - America/Creston\n* `America/Cuiaba` - America/Cuiaba\n* `America/Curacao` - America/Curacao\n* `America/Danmarkshavn` - America/Danmarkshavn\n* `America/Dawson` - America/Dawson\n* `America/Dawson_Creek` - America/Dawson_Creek\n* `America/Denver` - America/Denver\n* `America/Detroit` - America/Detroit\n* `America/Dominica` - America/Dominica\n* `America/Edmonton` - America/Edmonton\n* `America/Eirunepe` - America/Eirunepe\n* `America/El_Salvador` - America/El_Salvador\n* `America/Fort_Nelson` - America/Fort_Nelson\n* `America/Fortaleza` - America/Fortaleza\n* `America/Glace_Bay` - America/Glace_Bay\n* `America/Goose_Bay` - America/Goose_Bay\n* `America/Grand_Turk` - America/Grand_Turk\n* `America/Grenada` - America/Grenada\n* `America/Guadeloupe` - America/Guadeloupe\n* `America/Guatemala` - America/Guatemala\n* `America/Guayaquil` - America/Guayaquil\n* `America/Guyana` - America/Guyana\n* `America/Halifax` - America/Halifax\n* `America/Havana` - America/Havana\n* `America/Hermosillo` - America/Hermosillo\n* `America/Indiana/Indianapolis` - America/Indiana/Indianapolis\n* `America/Indiana/Knox` - America/Indiana/Knox\n* `America/Indiana/Marengo` - America/Indiana/Marengo\n* `America/Indiana/Petersburg` - America/Indiana/Petersburg\n* `America/Indiana/Tell_City` - America/Indiana/Tell_City\n* `America/Indiana/Vevay` - America/Indiana/Vevay\n* `America/Indiana/Vincennes` - America/Indiana/Vincennes\n* `America/Indiana/Winamac` - America/Indiana/Winamac\n* `America/Inuvik` - America/Inuvik\n* `America/Iqaluit` - America/Iqaluit\n* `America/Jamaica` - America/Jamaica\n* `America/Juneau` - America/Juneau\n* `America/Kentucky/Louisville` - America/Kentucky/Louisville\n* `America/Kentucky/Monticello` - America/Kentucky/Monticello\n* `America/Kralendijk` - America/Kralendijk\n* `America/La_Paz` - America/La_Paz\n* `America/Lima` - America/Lima\n* `America/Los_Angeles` - America/Los_Angeles\n* `America/Lower_Princes` - America/Lower_Princes\n* `America/Maceio` - America/Maceio\n* `America/Managua` - America/Managua\n* `America/Manaus` - America/Manaus\n* `America/Marigot` - America/Marigot\n* `America/Martinique` - America/Martinique\n* `America/Matamoros` - America/Matamoros\n* `America/Mazatlan` - America/Mazatlan\n* `America/Menominee` - America/Menominee\n* `America/Merida` - America/Merida\n* `America/Metlakatla` - America/Metlakatla\n* `America/Mexico_City` - America/Mexico_City\n* `America/Miquelon` - America/Miquelon\n* `America/Moncton` - America/Moncton\n* `America/Monterrey` - America/Monterrey\n* `America/Montevideo` - America/Montevideo\n* `America/Montserrat` - America/Montserrat\n* `America/Nassau` - America/Nassau\n* `America/New_York` - America/New_York\n* `America/Nome` - America/Nome\n* `America/Noronha` - America/Noronha\n* `America/North_Dakota/Beulah` - America/North_Dakota/Beulah\n* `America/North_Dakota/Center` - America/North_Dakota/Center\n* `America/North_Dakota/New_Salem` - America/North_Dakota/New_Salem\n* `America/Nuuk` - America/Nuuk\n* `America/Ojinaga` - America/Ojinaga\n* `America/Panama` - America/Panama\n* `America/Paramaribo` - America/Paramaribo\n* `America/Phoenix` - America/Phoenix\n* `America/Port-au-Prince` - America/Port-au-Prince\n* `America/Port_of_Spain` - America/Port_of_Spain\n* `America/Porto_Velho` - America/Porto_Velho\n* `America/Puerto_Rico` - America/Puerto_Rico\n* `America/Punta_Arenas` - America/Punta_Arenas\n* `America/Rankin_Inlet` - America/Rankin_Inlet\n* `America/Recife` - America/Recife\n* `America/Regina` - America/Regina\n* `America/Resolute` - America/Resolute\n* `America/Rio_Branco` - America/Rio_Branco\n* `America/Santarem` - America/Santarem\n* `America/Santiago` - America/Santiago\n* `America/Santo_Domingo` - America/Santo_Domingo\n* `America/Sao_Paulo` - America/Sao_Paulo\n* `America/Scoresbysund` - America/Scoresbysund\n* `America/Sitka` - America/Sitka\n* `America/St_Barthelemy` - America/St_Barthelemy\n* `America/St_Johns` - America/St_Johns\n* `America/St_Kitts` - America/St_Kitts\n* `America/St_Lucia` - America/St_Lucia\n* `America/St_Thomas` - America/St_Thomas\n* `America/St_Vincent` - America/St_Vincent\n* `America/Swift_Current` - America/Swift_Current\n* `America/Tegucigalpa` - America/Tegucigalpa\n* `America/Thule` - America/Thule\n* `America/Tijuana` - America/Tijuana\n* `America/Toronto` - America/Toronto\n* `America/Tortola` - America/Tortola\n* `America/Vancouver` - America/Vancouver\n* `America/Whitehorse` - America/Whitehorse\n* `America/Winnipeg` - America/Winnipeg\n* `America/Yakutat` - America/Yakutat\n* `Antarctica/Casey` - Antarctica/Casey\n* `Antarctica/Davis` - Antarctica/Davis\n* `Antarctica/DumontDUrville` - Antarctica/DumontDUrville\n* `Antarctica/Macquarie` - Antarctica/Macquarie\n* `Antarctica/Mawson` - Antarctica/Mawson\n* `Antarctica/McMurdo` - Antarctica/McMurdo\n* `Antarctica/Palmer` - Antarctica/Palmer\n* `Antarctica/Rothera` - Antarctica/Rothera\n* `Antarctica/Syowa` - Antarctica/Syowa\n* `Antarctica/Troll` - Antarctica/Troll\n* `Antarctica/Vostok` - Antarctica/Vostok\n* `Arctic/Longyearbyen` - Arctic/Longyearbyen\n* `Asia/Aden` - Asia/Aden\n* `Asia/Almaty` - Asia/Almaty\n* `Asia/Amman` - Asia/Amman\n* `Asia/Anadyr` - Asia/Anadyr\n* `Asia/Aqtau` - Asia/Aqtau\n* `Asia/Aqtobe` - Asia/Aqtobe\n* `Asia/Ashgabat` - Asia/Ashgabat\n* `Asia/Atyrau` - Asia/Atyrau\n* `Asia/Baghdad` - Asia/Baghdad\n* `Asia/Bahrain` - Asia/Bahrain\n* `Asia/Baku` - Asia/Baku\n* `Asia/Bangkok` - Asia/Bangkok\n* `Asia/Barnaul` - Asia/Barnaul\n* `Asia/Beirut` - Asia/Beirut\n* `Asia/Bishkek` - Asia/Bishkek\n* `Asia/Brunei` - Asia/Brunei\n* `Asia/Chita` - Asia/Chita\n* `Asia/Choibalsan` - Asia/Choibalsan\n* `Asia/Colombo` - Asia/Colombo\n* `Asia/Damascus` - Asia/Damascus\n* `Asia/Dhaka` - Asia/Dhaka\n* `Asia/Dili` - Asia/Dili\n* `Asia/Dubai` - Asia/Dubai\n* `Asia/Dushanbe` - Asia/Dushanbe\n* `Asia/Famagusta` - Asia/Famagusta\n* `Asia/Gaza` - Asia/Gaza\n* `Asia/Hebron` - Asia/Hebron\n* `Asia/Ho_Chi_Minh` - Asia/Ho_Chi_Minh\n* `Asia/Hong_Kong` - Asia/Hong_Kong\n* `Asia/Hovd` - Asia/Hovd\n* `Asia/Irkutsk` - Asia/Irkutsk\n* `Asia/Jakarta` - Asia/Jakarta\n* `Asia/Jayapura` - Asia/Jayapura\n* `Asia/Jerusalem` - Asia/Jerusalem\n* `Asia/Kabul` - Asia/Kabul\n* `Asia/Kamchatka` - Asia/Kamchatka\n* `Asia/Karachi` - Asia/Karachi\n* `Asia/Kathmandu` - Asia/Kathmandu\n* `Asia/Khandyga` - Asia/Khandyga\n* `Asia/Kolkata` - Asia/Kolkata\n* `Asia/Krasnoyarsk` - Asia/Krasnoyarsk\n* `Asia/Kuala_Lumpur` - Asia/Kuala_Lumpur\n* `Asia/Kuching` - Asia/Kuching\n* `Asia/Kuwait` - Asia/Kuwait\n* `Asia/Macau` - Asia/Macau\n* `Asia/Magadan` - Asia/Magadan\n* `Asia/Makassar` - Asia/Makassar\n* `Asia/Manila` - Asia/Manila\n* `Asia/Muscat` - Asia/Muscat\n* `Asia/Nicosia` - Asia/Nicosia\n* `Asia/Novokuznetsk` - Asia/Novokuznetsk\n* `Asia/Novosibirsk` - Asia/Novosibirsk\n* `Asia/Omsk` - Asia/Omsk\n* `Asia/Oral` - Asia/Oral\n* `Asia/Phnom_Penh` - Asia/Phnom_Penh\n* `Asia/Pontianak` - Asia/Pontianak\n* `Asia/Pyongyang` - Asia/Pyongyang\n* `Asia/Qatar` - Asia/Qatar\n* `Asia/Qostanay` - Asia/Qostanay\n* `Asia/Qyzylorda` - Asia/Qyzylorda\n* `Asia/Riyadh` - Asia/Riyadh\n* `Asia/Sakhalin` - Asia/Sakhalin\n* `Asia/Samarkand` - Asia/Samarkand\n* `Asia/Seoul` - Asia/Seoul\n* `Asia/Shanghai` - Asia/Shanghai\n* `Asia/Singapore` - Asia/Singapore\n* `Asia/Srednekolymsk` - Asia/Srednekolymsk\n* `Asia/Taipei` - Asia/Taipei\n* `Asia/Tashkent` - Asia/Tashkent\n* `Asia/Tbilisi` - Asia/Tbilisi\n* `Asia/Tehran` - Asia/Tehran\n* `Asia/Thimphu` - Asia/Thimphu\n* `Asia/Tokyo` - Asia/Tokyo\n* `Asia/Tomsk` - Asia/Tomsk\n* `Asia/Ulaanbaatar` - Asia/Ulaanbaatar\n* `Asia/Urumqi` - Asia/Urumqi\n* `Asia/Ust-Nera` - Asia/Ust-Nera\n* `Asia/Vientiane` - Asia/Vientiane\n* `Asia/Vladivostok` - Asia/Vladivostok\n* `Asia/Yakutsk` - Asia/Yakutsk\n* `Asia/Yangon` - Asia/Yangon\n* `Asia/Yekaterinburg` - Asia/Yekaterinburg\n* `Asia/Yerevan` - Asia/Yerevan\n* `Atlantic/Azores` - Atlantic/Azores\n* `Atlantic/Bermuda` - Atlantic/Bermuda\n* `Atlantic/Canary` - Atlantic/Canary\n* `Atlantic/Cape_Verde` - Atlantic/Cape_Verde\n* `Atlantic/Faroe` - Atlantic/Faroe\n* `Atlantic/Madeira` - Atlantic/Madeira\n* `Atlantic/Reykjavik` - Atlantic/Reykjavik\n* `Atlantic/South_Georgia` - Atlantic/South_Georgia\n* `Atlantic/St_Helena` - Atlantic/St_Helena\n* `Atlantic/Stanley` - Atlantic/Stanley\n* `Australia/Adelaide` - Australia/Adelaide\n* `Australia/Brisbane` - Australia/Brisbane\n* `Australia/Broken_Hill` - Australia/Broken_Hill\n* `Australia/Darwin` - Australia/Darwin\n* `Australia/Eucla` - Australia/Eucla\n* `Australia/Hobart` - Australia/Hobart\n* `Australia/Lindeman` - Australia/Lindeman\n* `Australia/Lord_Howe` - Australia/Lord_Howe\n* `Australia/Melbourne` - Australia/Melbourne\n* `Australia/Perth` - Australia/Perth\n* `Australia/Sydney` - Australia/Sydney\n* `Canada/Atlantic` - Canada/Atlantic\n* `Canada/Central` - Canada/Central\n* `Canada/Eastern` - Canada/Eastern\n* `Canada/Mountain` - Canada/Mountain\n* `Canada/Newfoundland` - Canada/Newfoundland\n* `Canada/Pacific` - Canada/Pacific\n* `Europe/Amsterdam` - Europe/Amsterdam\n* `Europe/Andorra` - Europe/Andorra\n* `Europe/Astrakhan` - Europe/Astrakhan\n* `Europe/Athens` - Europe/Athens\n* `Europe/Belgrade` - Europe/Belgrade\n* `Europe/Berlin` - Europe/Berlin\n* `Europe/Bratislava` - Europe/Bratislava\n* `Europe/Brussels` - Europe/Brussels\n* `Europe/Bucharest` - Europe/Bucharest\n* `Europe/Budapest` - Europe/Budapest\n* `Europe/Busingen` - Europe/Busingen\n* `Europe/Chisinau` - Europe/Chisinau\n* `Europe/Copenhagen` - Europe/Copenhagen\n* `Europe/Dublin` - Europe/Dublin\n* `Europe/Gibraltar` - Europe/Gibraltar\n* `Europe/Guernsey` - Europe/Guernsey\n* `Europe/Helsinki` - Europe/Helsinki\n* `Europe/Isle_of_Man` - Europe/Isle_of_Man\n* `Europe/Istanbul` - Europe/Istanbul\n* `Europe/Jersey` - Europe/Jersey\n* `Europe/Kaliningrad` - Europe/Kaliningrad\n* `Europe/Kirov` - Europe/Kirov\n* `Europe/Kyiv` - Europe/Kyiv\n* `Europe/Lisbon` - Europe/Lisbon\n* `Europe/Ljubljana` - Europe/Ljubljana\n* `Europe/London` - Europe/London\n* `Europe/Luxembourg` - Europe/Luxembourg\n* `Europe/Madrid` - Europe/Madrid\n* `Europe/Malta` - Europe/Malta\n* `Europe/Mariehamn` - Europe/Mariehamn\n* `Europe/Minsk` - Europe/Minsk\n* `Europe/Monaco` - Europe/Monaco\n* `Europe/Moscow` - Europe/Moscow\n* `Europe/Oslo` - Europe/Oslo\n* `Europe/Paris` - Europe/Paris\n* `Europe/Podgorica` - Europe/Podgorica\n* `Europe/Prague` - Europe/Prague\n* `Europe/Riga` - Europe/Riga\n* `Europe/Rome` - Europe/Rome\n* `Europe/Samara` - Europe/Samara\n* `Europe/San_Marino` - Europe/San_Marino\n* `Europe/Sarajevo` - Europe/Sarajevo\n* `Europe/Saratov` - Europe/Saratov\n* `Europe/Simferopol` - Europe/Simferopol\n* `Europe/Skopje` - Europe/Skopje\n* `Europe/Sofia` - Europe/Sofia\n* `Europe/Stockholm` - Europe/Stockholm\n* `Europe/Tallinn` - Europe/Tallinn\n* `Europe/Tirane` - Europe/Tirane\n* `Europe/Ulyanovsk` - Europe/Ulyanovsk\n* `Europe/Vaduz` - Europe/Vaduz\n* `Europe/Vatican` - Europe/Vatican\n* `Europe/Vienna` - Europe/Vienna\n* `Europe/Vilnius` - Europe/Vilnius\n* `Europe/Volgograd` - Europe/Volgograd\n* `Europe/Warsaw` - Europe/Warsaw\n* `Europe/Zagreb` - Europe/Zagreb\n* `Europe/Zurich` - Europe/Zurich\n* `GMT` - GMT\n* `Indian/Antananarivo` - Indian/Antananarivo\n* `Indian/Chagos` - Indian/Chagos\n* `Indian/Christmas` - Indian/Christmas\n* `Indian/Cocos` - Indian/Cocos\n* `Indian/Comoro` - Indian/Comoro\n* `Indian/Kerguelen` - Indian/Kerguelen\n* `Indian/Mahe` - Indian/Mahe\n* `Indian/Maldives` - Indian/Maldives\n* `Indian/Mauritius` - Indian/Mauritius\n* `Indian/Mayotte` - Indian/Mayotte\n* `Indian/Reunion` - Indian/Reunion\n* `Pacific/Apia` - Pacific/Apia\n* `Pacific/Auckland` - Pacific/Auckland\n* `Pacific/Bougainville` - Pacific/Bougainville\n* `Pacific/Chatham` - Pacific/Chatham\n* `Pacific/Chuuk` - Pacific/Chuuk\n* `Pacific/Easter` - Pacific/Easter\n* `Pacific/Efate` - Pacific/Efate\n* `Pacific/Fakaofo` - Pacific/Fakaofo\n* `Pacific/Fiji` - Pacific/Fiji\n* `Pacific/Funafuti` - Pacific/Funafuti\n* `Pacific/Galapagos` - Pacific/Galapagos\n* `Pacific/Gambier` - Pacific/Gambier\n* `Pacific/Guadalcanal` - Pacific/Guadalcanal\n* `Pacific/Guam` - Pacific/Guam\n* `Pacific/Honolulu` - Pacific/Honolulu\n* `Pacific/Kanton` - Pacific/Kanton\n* `Pacific/Kiritimati` - Pacific/Kiritimati\n* `Pacific/Kosrae` - Pacific/Kosrae\n* `Pacific/Kwajalein` - Pacific/Kwajalein\n* `Pacific/Majuro` - Pacific/Majuro\n* `Pacific/Marquesas` - Pacific/Marquesas\n* `Pacific/Midway` - Pacific/Midway\n* `Pacific/Nauru` - Pacific/Nauru\n* `Pacific/Niue` - Pacific/Niue\n* `Pacific/Norfolk` - Pacific/Norfolk\n* `Pacific/Noumea` - Pacific/Noumea\n* `Pacific/Pago_Pago` - Pacific/Pago_Pago\n* `Pacific/Palau` - Pacific/Palau\n* `Pacific/Pitcairn` - Pacific/Pitcairn\n* `Pacific/Pohnpei` - Pacific/Pohnpei\n* `Pacific/Port_Moresby` - Pacific/Port_Moresby\n* `Pacific/Rarotonga` - Pacific/Rarotonga\n* `Pacific/Saipan` - Pacific/Saipan\n* `Pacific/Tahiti` - Pacific/Tahiti\n* `Pacific/Tarawa` - Pacific/Tarawa\n* `Pacific/Tongatapu` - Pacific/Tongatapu\n* `Pacific/Wake` - Pacific/Wake\n* `Pacific/Wallis` - Pacific/Wallis\n* `US/Alaska` - US/Alaska\n* `US/Arizona` - US/Arizona\n* `US/Central` - US/Central\n* `US/Eastern` - US/Eastern\n* `US/Hawaii` - US/Hawaii\n* `US/Mountain` - US/Mountain\n* `US/Pacific` - US/Pacific\n* `UTC` - UTC")
	Enum("Africa/Abidjan", "Africa/Accra", "Africa/Addis_Ababa", "Africa/Algiers", "Africa/Asmara", "Africa/Bamako", "Africa/Bangui", "Africa/Banjul", "Africa/Bissau", "Africa/Blantyre", "Africa/Brazzaville", "Africa/Bujumbura", "Africa/Cairo", "Africa/Casablanca", "Africa/Ceuta", "Africa/Conakry", "Africa/Dakar", "Africa/Dar_es_Salaam", "Africa/Djibouti", "Africa/Douala", "Africa/El_Aaiun", "Africa/Freetown", "Africa/Gaborone", "Africa/Harare", "Africa/Johannesburg", "Africa/Juba", "Africa/Kampala", "Africa/Khartoum", "Africa/Kigali", "Africa/Kinshasa", "Africa/Lagos", "Africa/Libreville", "Africa/Lome", "Africa/Luanda", "Africa/Lubumbashi", "Africa/Lusaka", "Africa/Malabo", "Africa/Maputo", "Africa/Maseru", "Africa/Mbabane", "Africa/Mogadishu", "Africa/Monrovia", "Africa/Nairobi", "Africa/Ndjamena", "Africa/Niamey", "Africa/Nouakchott", "Africa/Ouagadougou", "Africa/Porto-Novo", "Africa/Sao_Tome", "Africa/Tripoli", "Africa/Tunis", "Africa/Windhoek", "America/Adak", "America/Anchorage", "America/Anguilla", "America/Antigua", "America/Araguaina", "America/Argentina/Buenos_Aires", "America/Argentina/Catamarca", "America/Argentina/Cordoba", "America/Argentina/Jujuy", "America/Argentina/La_Rioja", "America/Argentina/Mendoza", "America/Argentina/Rio_Gallegos", "America/Argentina/Salta", "America/Argentina/San_Juan", "America/Argentina/San_Luis", "America/Argentina/Tucuman", "America/Argentina/Ushuaia", "America/Aruba", "America/Asuncion", "America/Atikokan", "America/Bahia", "America/Bahia_Banderas", "America/Barbados", "America/Belem", "America/Belize", "America/Blanc-Sablon", "America/Boa_Vista", "America/Bogota", "America/Boise", "America/Cambridge_Bay", "America/Campo_Grande", "America/Cancun", "America/Caracas", "America/Cayenne", "America/Cayman", "America/Chicago", "America/Chihuahua", "America/Ciudad_Juarez", "America/Costa_Rica", "America/Creston", "America/Cuiaba", "America/Curacao", "America/Danmarkshavn", "America/Dawson", "America/Dawson_Creek", "America/Denver", "America/Detroit", "America/Dominica", "America/Edmonton", "America/Eirunepe", "America/El_Salvador", "America/Fort_Nelson", "America/Fortaleza", "America/Glace_Bay", "America/Goose_Bay", "America/Grand_Turk", "America/Grenada", "America/Guadeloupe", "America/Guatemala", "America/Guayaquil", "America/Guyana", "America/Halifax", "America/Havana", "America/Hermosillo", "America/Indiana/Indianapolis", "America/Indiana/Knox", "America/Indiana/Marengo", "America/Indiana/Petersburg", "America/Indiana/Tell_City", "America/Indiana/Vevay", "America/Indiana/Vincennes", "America/Indiana/Winamac", "America/Inuvik", "America/Iqaluit", "America/Jamaica", "America/Juneau", "America/Kentucky/Louisville", "America/Kentucky/Monticello", "America/Kralendijk", "America/La_Paz", "America/Lima", "America/Los_Angeles", "America/Lower_Princes", "America/Maceio", "America/Managua", "America/Manaus", "America/Marigot", "America/Martinique", "America/Matamoros", "America/Mazatlan", "America/Menominee", "America/Merida", "America/Metlakatla", "America/Mexico_City", "America/Miquelon", "America/Moncton", "America/Monterrey", "America/Montevideo", "America/Montserrat", "America/Nassau", "America/New_York", "America/Nome", "America/Noronha", "America/North_Dakota/Beulah", "America/North_Dakota/Center", "America/North_Dakota/New_Salem", "America/Nuuk", "America/Ojinaga", "America/Panama", "America/Paramaribo", "America/Phoenix", "America/Port-au-Prince", "America/Port_of_Spain", "America/Porto_Velho", "America/Puerto_Rico", "America/Punta_Arenas", "America/Rankin_Inlet", "America/Recife", "America/Regina", "America/Resolute", "America/Rio_Branco", "America/Santarem", "America/Santiago", "America/Santo_Domingo", "America/Sao_Paulo", "America/Scoresbysund", "America/Sitka", "America/St_Barthelemy", "America/St_Johns", "America/St_Kitts", "America/St_Lucia", "America/St_Thomas", "America/St_Vincent", "America/Swift_Current", "America/Tegucigalpa", "America/Thule", "America/Tijuana", "America/Toronto", "America/Tortola", "America/Vancouver", "America/Whitehorse", "America/Winnipeg", "America/Yakutat", "Antarctica/Casey", "Antarctica/Davis", "Antarctica/DumontDUrville", "Antarctica/Macquarie", "Antarctica/Mawson", "Antarctica/McMurdo", "Antarctica/Palmer", "Antarctica/Rothera", "Antarctica/Syowa", "Antarctica/Troll", "Antarctica/Vostok", "Arctic/Longyearbyen", "Asia/Aden", "Asia/Almaty", "Asia/Amman", "Asia/Anadyr", "Asia/Aqtau", "Asia/Aqtobe", "Asia/Ashgabat", "Asia/Atyrau", "Asia/Baghdad", "Asia/Bahrain", "Asia/Baku", "Asia/Bangkok", "Asia/Barnaul", "Asia/Beirut", "Asia/Bishkek", "Asia/Brunei", "Asia/Chita", "Asia/Choibalsan", "Asia/Colombo", "Asia/Damascus", "Asia/Dhaka", "Asia/Dili", "Asia/Dubai", "Asia/Dushanbe", "Asia/Famagusta", "Asia/Gaza", "Asia/Hebron", "Asia/Ho_Chi_Minh", "Asia/Hong_Kong", "Asia/Hovd", "Asia/Irkutsk", "Asia/Jakarta", "Asia/Jayapura", "Asia/Jerusalem", "Asia/Kabul", "Asia/Kamchatka", "Asia/Karachi", "Asia/Kathmandu", "Asia/Khandyga", "Asia/Kolkata", "Asia/Krasnoyarsk", "Asia/Kuala_Lumpur", "Asia/Kuching", "Asia/Kuwait", "Asia/Macau", "Asia/Magadan", "Asia/Makassar", "Asia/Manila", "Asia/Muscat", "Asia/Nicosia", "Asia/Novokuznetsk", "Asia/Novosibirsk", "Asia/Omsk", "Asia/Oral", "Asia/Phnom_Penh", "Asia/Pontianak", "Asia/Pyongyang", "Asia/Qatar", "Asia/Qostanay", "Asia/Qyzylorda", "Asia/Riyadh", "Asia/Sakhalin", "Asia/Samarkand", "Asia/Seoul", "Asia/Shanghai", "Asia/Singapore", "Asia/Srednekolymsk", "Asia/Taipei", "Asia/Tashkent", "Asia/Tbilisi", "Asia/Tehran", "Asia/Thimphu", "Asia/Tokyo", "Asia/Tomsk", "Asia/Ulaanbaatar", "Asia/Urumqi", "Asia/Ust-Nera", "Asia/Vientiane", "Asia/Vladivostok", "Asia/Yakutsk", "Asia/Yangon", "Asia/Yekaterinburg", "Asia/Yerevan", "Atlantic/Azores", "Atlantic/Bermuda", "Atlantic/Canary", "Atlantic/Cape_Verde", "Atlantic/Faroe", "Atlantic/Madeira", "Atlantic/Reykjavik", "Atlantic/South_Georgia", "Atlantic/St_Helena", "Atlantic/Stanley", "Australia/Adelaide", "Australia/Brisbane", "Australia/Broken_Hill", "Australia/Darwin", "Australia/Eucla", "Australia/Hobart", "Australia/Lindeman", "Australia/Lord_Howe", "Australia/Melbourne", "Australia/Perth", "Australia/Sydney", "Canada/Atlantic", "Canada/Central", "Canada/Eastern", "Canada/Mountain", "Canada/Newfoundland", "Canada/Pacific", "Europe/Amsterdam", "Europe/Andorra", "Europe/Astrakhan", "Europe/Athens", "Europe/Belgrade", "Europe/Berlin", "Europe/Bratislava", "Europe/Brussels", "Europe/Bucharest", "Europe/Budapest", "Europe/Busingen", "Europe/Chisinau", "Europe/Copenhagen", "Europe/Dublin", "Europe/Gibraltar", "Europe/Guernsey", "Europe/Helsinki", "Europe/Isle_of_Man", "Europe/Istanbul", "Europe/Jersey", "Europe/Kaliningrad", "Europe/Kirov", "Europe/Kyiv", "Europe/Lisbon", "Europe/Ljubljana", "Europe/London", "Europe/Luxembourg", "Europe/Madrid", "Europe/Malta", "Europe/Mariehamn", "Europe/Minsk", "Europe/Monaco", "Europe/Moscow", "Europe/Oslo", "Europe/Paris", "Europe/Podgorica", "Europe/Prague", "Europe/Riga", "Europe/Rome", "Europe/Samara", "Europe/San_Marino", "Europe/Sarajevo", "Europe/Saratov", "Europe/Simferopol", "Europe/Skopje", "Europe/Sofia", "Europe/Stockholm", "Europe/Tallinn", "Europe/Tirane", "Europe/Ulyanovsk", "Europe/Vaduz", "Europe/Vatican", "Europe/Vienna", "Europe/Vilnius", "Europe/Volgograd", "Europe/Warsaw", "Europe/Zagreb", "Europe/Zurich", "GMT", "Indian/Antananarivo", "Indian/Chagos", "Indian/Christmas", "Indian/Cocos", "Indian/Comoro", "Indian/Kerguelen", "Indian/Mahe", "Indian/Maldives", "Indian/Mauritius", "Indian/Mayotte", "Indian/Reunion", "Pacific/Apia", "Pacific/Auckland", "Pacific/Bougainville", "Pacific/Chatham", "Pacific/Chuuk", "Pacific/Easter", "Pacific/Efate", "Pacific/Fakaofo", "Pacific/Fiji", "Pacific/Funafuti", "Pacific/Galapagos", "Pacific/Gambier", "Pacific/Guadalcanal", "Pacific/Guam", "Pacific/Honolulu", "Pacific/Kanton", "Pacific/Kiritimati", "Pacific/Kosrae", "Pacific/Kwajalein", "Pacific/Majuro", "Pacific/Marquesas", "Pacific/Midway", "Pacific/Nauru", "Pacific/Niue", "Pacific/Norfolk", "Pacific/Noumea", "Pacific/Pago_Pago", "Pacific/Palau", "Pacific/Pitcairn", "Pacific/Pohnpei", "Pacific/Port_Moresby", "Pacific/Rarotonga", "Pacific/Saipan", "Pacific/Tahiti", "Pacific/Tarawa", "Pacific/Tongatapu", "Pacific/Wake", "Pacific/Wallis", "US/Alaska", "US/Arizona", "US/Central", "US/Eastern", "US/Hawaii", "US/Mountain", "US/Pacific", "UTC")
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
	Meta("openapi:tag:search")
	Meta("openapi:tag:work-items")
	Meta("openapi:tag:cycles")
	Meta("openapi:tag:intake")
	Meta("openapi:tag:modules")
	Meta("openapi:tag:draft-work-items")
	Meta("openapi:tag:workspaces")
	Meta("openapi:tag:projects")
	HTTP(func() {
		Consumes("application/json", "application/x-www-form-urlencoded", "multipart/form-data")
		Produces("application/json")
	})
})

var _ = Service("WorkItemCore", func() {
	HTTP(func() { RuntimeCORS() })
	Method("GetAPIV1WorkspacesBySlugIssuesSearch", func() {
		Description("Perform semantic search across issue names, sequence IDs, and project identifiers.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_issues_search")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/issues/search/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueSearchEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("limit", String, func() {
				Description("Maximum number of results to return")
				Meta("openapi:allowEmptyValue", "false")
				Meta("openapi:format", "")
			})
			Attribute("projectID", String, func() {
				Description("Project ID for filtering results within a specific project")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("search", String, func() {
				Description("Search query to filter results by name, description, or identifier")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			Attribute("workspaceSearch", String, func() {
				Description("Whether to search across entire workspace or within specific project")
				Meta("openapi:allowEmptyValue", "false")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:search")
			GET("/api/v1/workspaces/{slug}/issues/search/")
			Param("limit")
			Param("projectID:project_id")
			Param("search")
			Param("slug")
			Param("workspaceSearch:workspace_search")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("Work item search results")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIssueIssuesearchserializerResponse)
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugIssuesProjectIdentifierIssueIdentifier", func() {
		Description("Retrieve a specific work item using workspace slug, project identifier, and issue identifier.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_issues__project_identifier___issue_identifier")
		Meta("openapi:summary", "Retrieve work item by identifiers")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/issues/{project_identifier}-{issue_identifier}/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.WorkspaceIssueAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("identifier", String, func() {
				Description("Project identifier and issue sequence in one segment; Django lookup preserves the complete spelling.")
				Example("IC-1")
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("identifier", "slug")
		})
		Result(workItemResponse)
		workItemErrors()

		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			GET("/api/v1/workspaces/{slug}/issues/{identifier}/")
			Param("identifier")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("Work item details")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIssueIssueserializerResponse, func() {
					Meta("openapi:allOf:reference", "true")
					Example("Issue", Val{"assignees": []any{"550e8400-e29b-41d4-a716-446655440001"}, "created_at": "2024-01-15T10:30:00Z", "description": "Add OAuth 2.0 authentication flow", "id": "550e8400-e29b-41d4-a716-446655440000", "labels": []any{"550e8400-e29b-41d4-a716-446655440002"}, "name": "Implement user authentication", "priority": "high", "sequence_id": 1, "updated_at": "2024-01-15T10:30:00Z"})
				})
			})
		})
	})
	Method("PostAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssues", func() {
		Description("Assign multiple work items to a cycle. Automatically handles bulk creation and updates with activity tracking.")
		Meta("openapi:operationId", "post_api_v1_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_cycle_issues")
		Meta("openapi:summary", "Add Work Items to Cycle")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/v1/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.cycle.CycleIssueListCreateAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("cycle_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("cycle_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()

		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:cycles")
			POST("/api/v1/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/")
			Param("cycle_id")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAPISerializersCycleCycleissuerequestserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true, func() {
				Meta("openapi:allOf:reference", "true")
				Example("CycleIssueRequestSerializer", func() {
					Description("Example request for adding cycle issues")
					Value(Val{"issues": []any{"0ec6cfa4-e906-4aad-9390-2df0303a41cd", "0ec6cfa4-e906-4aad-9390-2df0303a41ce"}})
				})
			})
			Response(200, func() {
				Description("Cycle work items added")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersCycleCycleissueserializerResponse, func() {
					Meta("openapi:allOf:reference", "true")
					Example("CycleIssue", Val{"created_at": "2024-01-01T10:30:00Z", "cycle": "550e8400-e29b-41d4-a716-446655440001", "id": "550e8400-e29b-41d4-a716-446655440000", "issue": "550e8400-e29b-41d4-a716-446655440002", "sub_issues_count": 3, "updated_at": "2024-01-10T15:45:00Z"})
				})
			})
		})
	})
	Method("DeleteAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesByIssueID", func() {
		Description("Remove a work item from a cycle while keeping the work item in the project.")
		Meta("openapi:operationId", "delete_api_v1_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_cycle_issues_by_issue_id")
		Meta("openapi:summary", "Delete cycle work item")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/v1/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/{issue_id}/\"")
		Meta("openapi:extension:x-plane-handler", "\"delete\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.cycle.CycleIssueDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("cycle_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("issue_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("cycle_id", "issue_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:cycles")
			DELETE("/api/v1/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/{issue_id}/")
			Param("cycle_id")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(204, func() {
				Description("Resource deleted successfully")
			})
		})
	})
	Method("PostAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDTransferIssues", func() {
		Description("Move incomplete work items from the current cycle to a new target cycle. Captures progress snapshot and transfers only unfinished work items.")
		Meta("openapi:operationId", "post_api_v1_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_transfer_issues")
		Meta("openapi:summary", "Transfer cycle work items")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/v1/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/transfer-issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.cycle.TransferCycleIssueAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("cycle_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("cycle_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()

		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:cycles")
			POST("/api/v1/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/transfer-issues/")
			Param("cycle_id")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAPISerializersCycleTransfercycleissuerequestserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true, func() {
				Meta("openapi:allOf:reference", "true")
				Example("TransferCycleIssueRequestSerializer", func() {
					Description("Example request for transferring cycle issues")
					Value(Val{"new_cycle_id": "0ec6cfa4-e906-4aad-9390-2df0303a41ce"})
				})
			})
			Response(200, func() {
				Description("Work items transferred successfully")
				ContentType("application/json")
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDIntakeIssues", func() {
		Description("Retrieve all work items in the project's intake queue. Returns paginated results when listing all intake work items.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_intake_issues")
		Meta("openapi:summary", "List intake work items")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/intake-issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectLitePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.intake.IntakeIssueListCreateAPIEndpoint\"")
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
			Attribute("perPage", String, func() {
				Description("Number of results per page (default: 20, max: 100)")
				Meta("openapi:allowEmptyValue", "false")
				Meta("openapi:format", "")
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:intake")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/intake-issues/")
			Param("cursor")
			Param("expand")
			Param("fields")
			Param("perPage:per_page")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("Paginated list of intake work items")
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersPaginatedintakeissueresponseResponse)
			})
		})
	})
	Method("PostAPIV1WorkspacesBySlugProjectsByProjectIDIntakeIssues", func() {
		Description("Submit a new work item to the project's intake queue for review and triage. Automatically creates the work item with default triage state and tracks activity.")
		Meta("openapi:operationId", "post_api_v1_workspaces_by_slug_projects_by_project_id_intake_issues")
		Meta("openapi:summary", "Create intake work item")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/v1/workspaces/{slug}/projects/{project_id}/intake-issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectLitePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.intake.IntakeIssueListCreateAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:intake")
			POST("/api/v1/workspaces/{slug}/projects/{project_id}/intake-issues/")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAPISerializersIntakeIntakeissuecreateserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(201, func() {
				Description("Intake work item created")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIntakeIntakeissueserializerResponse)
			})
		})
	})
	Method("DeleteAPIV1WorkspacesBySlugProjectsByProjectIDIntakeIssuesByIssueID", func() {
		Description("Permanently remove an intake work item from the triage queue. Also deletes the underlying work item if it hasn't been accepted yet.")
		Meta("openapi:operationId", "delete_api_v1_workspaces_by_slug_projects_by_project_id_intake_issues_by_issue_id")
		Meta("openapi:summary", "Delete intake work item")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/v1/workspaces/{slug}/projects/{project_id}/intake-issues/{issue_id}/\"")
		Meta("openapi:extension:x-plane-handler", "\"delete\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectLitePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.intake.IntakeIssueDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:intake")
			DELETE("/api/v1/workspaces/{slug}/projects/{project_id}/intake-issues/{issue_id}/")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(204, func() {
				Description("Resource deleted successfully")
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDIntakeIssuesByIssueID", func() {
		Description("Retrieve details of a specific intake work item.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_intake_issues_by_issue_id")
		Meta("openapi:summary", "Retrieve intake work item")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/intake-issues/{issue_id}/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectLitePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.intake.IntakeIssueDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:intake")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/intake-issues/{issue_id}/")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("Intake work item")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIntakeIntakeissueserializerResponse)
			})
		})
	})
	Method("PatchAPIV1WorkspacesBySlugProjectsByProjectIDIntakeIssuesByIssueID", func() {
		Description("Modify an existing intake work item's properties or status for triage processing. Supports status changes like accept, reject, or mark as duplicate.")
		Meta("openapi:operationId", "patch_api_v1_workspaces_by_slug_projects_by_project_id_intake_issues_by_issue_id")
		Meta("openapi:summary", "Update intake work item")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/v1/workspaces/{slug}/projects/{project_id}/intake-issues/{issue_id}/\"")
		Meta("openapi:extension:x-plane-handler", "\"patch\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectLitePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.intake.IntakeIssueDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:intake")
			PATCH("/api/v1/workspaces/{slug}/projects/{project_id}/intake-issues/{issue_id}/")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAPISerializersIntakeIntakeissueupdateserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				Description("Intake work item updated")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIntakeIntakeissueserializerResponse)
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDIssues", func() {
		Description("Retrieve a paginated list of all work items in a project. Supports filtering, ordering, and field selection through query parameters.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_issues")
		Meta("openapi:summary", "List work items")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueListCreateAPIEndpoint\"")
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
			Attribute("externalID", String, func() {
				Description("External system identifier for filtering or lookup")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("externalSource", String, func() {
				Description("External system source name for filtering or lookup")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("fields", String, func() {
				Description("Comma-separated list of fields to include in response")
				Meta("openapi:allowEmptyValue", "false")
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
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/issues/")
			Param("cursor")
			Param("expand")
			Param("externalID:external_id")
			Param("externalSource:external_source")
			Param("fields")
			Param("orderBy:order_by")
			Param("perPage:per_page")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("Paginated list of work items")
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersPaginatedworkitemresponseResponse)
			})
		})
	})
	Method("PostAPIV1WorkspacesBySlugProjectsByProjectIDIssues", func() {
		Description("Create a new work item in the specified project with the provided details.")
		Meta("openapi:operationId", "post_api_v1_workspaces_by_slug_projects_by_project_id_issues")
		Meta("openapi:summary", "Create work item")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/v1/workspaces/{slug}/projects/{project_id}/issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueListCreateAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()

		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			POST("/api/v1/workspaces/{slug}/projects/{project_id}/issues/")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAPISerializersIssueIssueserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true, func() {
				Meta("openapi:allOf:reference", "true")
				Example("IssueCreateSerializer", func() {
					Description("Example request for creating a work item")
					Value(Val{"assignees": []any{"0ec6cfa4-e906-4aad-9390-2df0303a41cd"}, "description": "New issue description", "external_id": "1234567890", "external_source": "github", "labels": []any{"0ec6cfa4-e906-4aad-9390-2df0303a41ce"}, "name": "New Issue", "priority": "medium", "state": "0ec6cfa4-e906-4aad-9390-2df0303a41cd"})
				})
			})
			Response(201, func() {
				Description("Work Item created successfully")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIssueIssueserializerResponse, func() {
					Meta("openapi:allOf:reference", "true")
					Example("Issue", Val{"assignees": []any{"550e8400-e29b-41d4-a716-446655440001"}, "created_at": "2024-01-15T10:30:00Z", "description": "Add OAuth 2.0 authentication flow", "id": "550e8400-e29b-41d4-a716-446655440000", "labels": []any{"550e8400-e29b-41d4-a716-446655440002"}, "name": "Implement user authentication", "priority": "high", "sequence_id": 1, "updated_at": "2024-01-15T10:30:00Z"})
				})
			})
		})
	})
	Method("DeleteAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByPk", func() {
		Description("Permanently delete an existing work item from the project. Only admins or the item creator can perform this action.")
		Meta("openapi:operationId", "delete_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_pk")
		Meta("openapi:summary", "Delete work item")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/v1/workspaces/{slug}/projects/{project_id}/issues/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"delete\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("pk", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			DELETE("/api/v1/workspaces/{slug}/projects/{project_id}/issues/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(204, func() {
				Description("Resource deleted successfully")
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByPk", func() {
		Description("Retrieve details of a specific work item.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_pk")
		Meta("openapi:summary", "Retrieve work item")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/issues/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("expand", String, func() {
				Description("Comma-separated list of related fields to expand in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("externalID", String, func() {
				Description("External system identifier for filtering or lookup")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("externalSource", String, func() {
				Description("External system source name for filtering or lookup")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("fields", String, func() {
				Description("Comma-separated list of fields to include in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("orderBy", String, func() {
				Description("Field to order results by. Prefix with '-' for descending order")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("pk", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()

		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/issues/{pk}/")
			Param("expand")
			Param("externalID:external_id")
			Param("externalSource:external_source")
			Param("fields")
			Param("orderBy:order_by")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("List of issues or issue details")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIssueIssueserializerResponse, func() {
					Meta("openapi:allOf:reference", "true")
					Example("Issue", Val{"assignees": []any{"550e8400-e29b-41d4-a716-446655440001"}, "created_at": "2024-01-15T10:30:00Z", "description": "Add OAuth 2.0 authentication flow", "id": "550e8400-e29b-41d4-a716-446655440000", "labels": []any{"550e8400-e29b-41d4-a716-446655440002"}, "name": "Implement user authentication", "priority": "high", "sequence_id": 1, "updated_at": "2024-01-15T10:30:00Z"})
				})
			})
		})
	})
	Method("PatchAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByPk", func() {
		Description("Partially update an existing work item with the provided fields. Supports external ID validation to prevent conflicts.")
		Meta("openapi:operationId", "patch_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_pk")
		Meta("openapi:summary", "Partially update work item")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/v1/workspaces/{slug}/projects/{project_id}/issues/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"patch\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("pk", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()

		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			PATCH("/api/v1/workspaces/{slug}/projects/{project_id}/issues/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAPISerializersIssueIssueserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false, func() {
				Meta("openapi:allOf:reference", "true")
				Example("IssueUpdateSerializer", func() {
					Description("Example request for updating a work item")
					Value(Val{"assignees": []any{"0ec6cfa4-e906-4aad-9390-2df0303a41cd"}, "description": "Updated issue description", "labels": []any{"0ec6cfa4-e906-4aad-9390-2df0303a41ce"}, "name": "Updated Issue", "priority": "medium", "state": "0ec6cfa4-e906-4aad-9390-2df0303a41cd"})
				})
			})
			Response(200, func() {
				Description("Work Item patched successfully")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIssueIssueserializerResponse, func() {
					Meta("openapi:allOf:reference", "true")
					Example("Issue", Val{"assignees": []any{"550e8400-e29b-41d4-a716-446655440001"}, "created_at": "2024-01-15T10:30:00Z", "description": "Add OAuth 2.0 authentication flow", "id": "550e8400-e29b-41d4-a716-446655440000", "labels": []any{"550e8400-e29b-41d4-a716-446655440002"}, "name": "Implement user authentication", "priority": "high", "sequence_id": 1, "updated_at": "2024-01-15T10:30:00Z"})
				})
			})
		})
	})
	Method("PostAPIV1WorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleIssues", func() {
		Description("Assign multiple work items to a module or move them from another module. Automatically handles bulk creation and updates with activity tracking.")
		Meta("openapi:operationId", "post_api_v1_workspaces_by_slug_projects_by_project_id_modules_by_module_id_module_issues")
		Meta("openapi:summary", "Add Work Items to Module")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/v1/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.module.ModuleIssueListCreateAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("module_id", String, func() {
				Description("Module ID")
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("module_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()

		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:modules")
			POST("/api/v1/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-issues/")
			Param("module_id")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAPISerializersModuleModuleissuerequestserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true, func() {
				Meta("openapi:allOf:reference", "true")
				Example("ModuleIssueRequestSerializer", func() {
					Description("Example request for adding module issues")
					Value(Val{"issues": []any{"0ec6cfa4-e906-4aad-9390-2df0303a41cd", "0ec6cfa4-e906-4aad-9390-2df0303a41ce"}})
				})
			})
			Response(200, func() {
				Description("Module issues added")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersModuleModuleissueserializerResponse, func() {
					Meta("openapi:allOf:reference", "true")
					Example("ModuleIssue", Val{"created_at": "2024-01-01T10:30:00Z", "id": "550e8400-e29b-41d4-a716-446655440000", "issue": "550e8400-e29b-41d4-a716-446655440002", "module": "550e8400-e29b-41d4-a716-446655440001", "sub_issues_count": 2, "updated_at": "2024-01-10T15:45:00Z"})
				})
			})
		})
	})
	Method("DeleteAPIV1WorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleIssuesByIssueID", func() {
		Description("Remove a work item from a module while keeping the work item in the project.")
		Meta("openapi:operationId", "delete_api_v1_workspaces_by_slug_projects_by_project_id_modules_by_module_id_module_issues_by_issue_id")
		Meta("openapi:summary", "Delete module work item")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/v1/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-issues/{issue_id}/\"")
		Meta("openapi:extension:x-plane-handler", "\"delete\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.module.ModuleIssueDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				Format(FormatUUID)
			})
			Attribute("module_id", String, func() {
				Description("Module ID")
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "module_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:modules")
			DELETE("/api/v1/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-issues/{issue_id}/")
			Param("issue_id")
			Param("module_id")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(204, func() {
				Description("Resource deleted successfully")
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItems", func() {
		Description("Retrieve a paginated list of all work items in a project. Supports filtering, ordering, and field selection through query parameters.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_work_items")
		Meta("openapi:summary", "List work items")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/work-items/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueListCreateAPIEndpoint\"")
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
			Attribute("externalID", String, func() {
				Description("External system identifier for filtering or lookup")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("externalSource", String, func() {
				Description("External system source name for filtering or lookup")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("fields", String, func() {
				Description("Comma-separated list of fields to include in response")
				Meta("openapi:allowEmptyValue", "false")
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
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/")
			Param("cursor")
			Param("expand")
			Param("externalID:external_id")
			Param("externalSource:external_source")
			Param("fields")
			Param("orderBy:order_by")
			Param("perPage:per_page")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("Paginated list of work items")
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersPaginatedworkitemresponseResponse)
			})
		})
	})
	Method("PostAPIV1WorkspacesBySlugProjectsByProjectIDWorkItems", func() {
		Description("Create a new work item in the specified project with the provided details.")
		Meta("openapi:operationId", "post_api_v1_workspaces_by_slug_projects_by_project_id_work_items")
		Meta("openapi:summary", "Create work item")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/v1/workspaces/{slug}/projects/{project_id}/work-items/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueListCreateAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()

		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			POST("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAPISerializersIssueIssueserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true, func() {
				Meta("openapi:allOf:reference", "true")
				Example("IssueCreateSerializer", func() {
					Description("Example request for creating a work item")
					Value(Val{"assignees": []any{"0ec6cfa4-e906-4aad-9390-2df0303a41cd"}, "description": "New issue description", "external_id": "1234567890", "external_source": "github", "labels": []any{"0ec6cfa4-e906-4aad-9390-2df0303a41ce"}, "name": "New Issue", "priority": "medium", "state": "0ec6cfa4-e906-4aad-9390-2df0303a41cd"})
				})
			})
			Response(201, func() {
				Description("Work Item created successfully")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIssueIssueserializerResponse, func() {
					Meta("openapi:allOf:reference", "true")
					Example("Issue", Val{"assignees": []any{"550e8400-e29b-41d4-a716-446655440001"}, "created_at": "2024-01-15T10:30:00Z", "description": "Add OAuth 2.0 authentication flow", "id": "550e8400-e29b-41d4-a716-446655440000", "labels": []any{"550e8400-e29b-41d4-a716-446655440002"}, "name": "Implement user authentication", "priority": "high", "sequence_id": 1, "updated_at": "2024-01-15T10:30:00Z"})
				})
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDRelations", func() {
		Description("Retrieve all relationships for a work item including blocking, blocked_by, duplicate, relates_to, start_before, start_after, finish_before, and finish_after relations.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_relations")
		Meta("openapi:summary", "List work item relations")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/relations/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueRelationListCreateAPIEndpoint\"")
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
				Format(FormatUUID)
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
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()

		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/relations/")
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
				Description("Work item relations grouped by relation type")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIssueIssuerelationresponseserializerResponse, func() {
					Meta("openapi:allOf:reference", "true")
					Example("WorkItemRelationsResponse", func() {
						Meta("openapi:example:summary", "Work Item Relations Response")
						Value(Val{"blocked_by": []any{Val{"issue_id": "550e8400-e29b-41d4-a716-446655440002", "project_id": "550e8400-e29b-41d4-a716-446655440011"}}, "blocking": []any{Val{"issue_id": "550e8400-e29b-41d4-a716-446655440000", "project_id": "550e8400-e29b-41d4-a716-446655440010"}, Val{"issue_id": "550e8400-e29b-41d4-a716-446655440001", "project_id": "550e8400-e29b-41d4-a716-446655440010"}}, "duplicate": []any{}, "finish_after": []any{}, "finish_before": []any{}, "relates_to": []any{Val{"issue_id": "550e8400-e29b-41d4-a716-446655440003", "project_id": "550e8400-e29b-41d4-a716-446655440010"}}, "start_after": []any{}, "start_before": []any{Val{"issue_id": "550e8400-e29b-41d4-a716-446655440004", "project_id": "550e8400-e29b-41d4-a716-446655440012"}}})
					})
				})
			})
		})
	})
	Method("PostAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDRelations", func() {
		Description("Create relationships between work items. Supports various relation types including blocking, blocked_by, duplicate, relates_to, start_before, start_after, finish_before, and finish_after.")
		Meta("openapi:operationId", "post_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_relations")
		Meta("openapi:summary", "Create work item relation")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/relations/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueRelationListCreateAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			POST("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/relations/")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAPISerializersIssueIssuerelationcreateserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true, func() {
				Meta("openapi:allOf:reference", "true")
				Example("CreateBlockingRelation", func() {
					Meta("openapi:example:summary", "Create blocking relation")
					Value(Val{"issues": []any{"550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440001"}, "relation_type": "blocking"})
				})
			})
			Response(201, func() {
				Description("Work item relations created successfully")
				ContentType("application/json")
			})
		})
	})
	Method("DeleteAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByPk", func() {
		Description("Permanently delete an existing work item from the project. Only admins or the item creator can perform this action.")
		Meta("openapi:operationId", "delete_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_pk")
		Meta("openapi:summary", "Delete work item")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/v1/workspaces/{slug}/projects/{project_id}/work-items/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"delete\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("pk", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			DELETE("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(204, func() {
				Description("Resource deleted successfully")
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByPk", func() {
		Description("Retrieve details of a specific work item.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_pk")
		Meta("openapi:summary", "Retrieve work item")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/work-items/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("expand", String, func() {
				Description("Comma-separated list of related fields to expand in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("externalID", String, func() {
				Description("External system identifier for filtering or lookup")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("externalSource", String, func() {
				Description("External system source name for filtering or lookup")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("fields", String, func() {
				Description("Comma-separated list of fields to include in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("orderBy", String, func() {
				Description("Field to order results by. Prefix with '-' for descending order")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("pk", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()

		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{pk}/")
			Param("expand")
			Param("externalID:external_id")
			Param("externalSource:external_source")
			Param("fields")
			Param("orderBy:order_by")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("List of issues or issue details")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIssueIssueserializerResponse, func() {
					Meta("openapi:allOf:reference", "true")
					Example("Issue", Val{"assignees": []any{"550e8400-e29b-41d4-a716-446655440001"}, "created_at": "2024-01-15T10:30:00Z", "description": "Add OAuth 2.0 authentication flow", "id": "550e8400-e29b-41d4-a716-446655440000", "labels": []any{"550e8400-e29b-41d4-a716-446655440002"}, "name": "Implement user authentication", "priority": "high", "sequence_id": 1, "updated_at": "2024-01-15T10:30:00Z"})
				})
			})
		})
	})
	Method("PatchAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByPk", func() {
		Description("Partially update an existing work item with the provided fields. Supports external ID validation to prevent conflicts.")
		Meta("openapi:operationId", "patch_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_pk")
		Meta("openapi:summary", "Partially update work item")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/v1/workspaces/{slug}/projects/{project_id}/work-items/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"patch\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("pk", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()

		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			PATCH("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAPISerializersIssueIssueserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false, func() {
				Meta("openapi:allOf:reference", "true")
				Example("IssueUpdateSerializer", func() {
					Description("Example request for updating a work item")
					Value(Val{"assignees": []any{"0ec6cfa4-e906-4aad-9390-2df0303a41cd"}, "description": "Updated issue description", "labels": []any{"0ec6cfa4-e906-4aad-9390-2df0303a41ce"}, "name": "Updated Issue", "priority": "medium", "state": "0ec6cfa4-e906-4aad-9390-2df0303a41cd"})
				})
			})
			Response(200, func() {
				Description("Work Item patched successfully")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIssueIssueserializerResponse, func() {
					Meta("openapi:allOf:reference", "true")
					Example("Issue", Val{"assignees": []any{"550e8400-e29b-41d4-a716-446655440001"}, "created_at": "2024-01-15T10:30:00Z", "description": "Add OAuth 2.0 authentication flow", "id": "550e8400-e29b-41d4-a716-446655440000", "labels": []any{"550e8400-e29b-41d4-a716-446655440002"}, "name": "Implement user authentication", "priority": "high", "sequence_id": 1, "updated_at": "2024-01-15T10:30:00Z"})
				})
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugWorkItemsSearch", func() {
		Description("Perform semantic search across issue names, sequence IDs, and project identifiers.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_work_items_search")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/work-items/search/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueSearchEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("limit", String, func() {
				Description("Maximum number of results to return")
				Meta("openapi:allowEmptyValue", "false")
				Meta("openapi:format", "")
			})
			Attribute("projectID", String, func() {
				Description("Project ID for filtering results within a specific project")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("search", String, func() {
				Description("Search query to filter results by name, description, or identifier")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			Attribute("workspaceSearch", String, func() {
				Description("Whether to search across entire workspace or within specific project")
				Meta("openapi:allowEmptyValue", "false")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:search")
			GET("/api/v1/workspaces/{slug}/work-items/search/")
			Param("limit")
			Param("projectID:project_id")
			Param("search")
			Param("slug")
			Param("workspaceSearch:workspace_search")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("Work item search results")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIssueIssuesearchserializerResponse)
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugWorkItemsProjectIdentifierIssueIdentifier", func() {
		Description("Retrieve a specific work item using workspace slug, project identifier, and issue identifier.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_work_items__project_identifier___issue_identifier")
		Meta("openapi:summary", "Retrieve work item by identifiers")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/work-items/{project_identifier}-{issue_identifier}/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.WorkspaceIssueAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("identifier", String, func() {
				Description("Project identifier and issue sequence in one segment; Django lookup preserves the complete spelling.")
				Example("IC-1")
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("identifier", "slug")
		})
		Result(workItemResponse)
		workItemErrors()

		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			GET("/api/v1/workspaces/{slug}/work-items/{identifier}/")
			Param("identifier")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("Work item details")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersIssueIssueserializerResponse, func() {
					Meta("openapi:allOf:reference", "true")
					Example("Issue", Val{"assignees": []any{"550e8400-e29b-41d4-a716-446655440001"}, "created_at": "2024-01-15T10:30:00Z", "description": "Add OAuth 2.0 authentication flow", "id": "550e8400-e29b-41d4-a716-446655440000", "labels": []any{"550e8400-e29b-41d4-a716-446655440002"}, "name": "Implement user authentication", "priority": "high", "sequence_id": 1, "updated_at": "2024-01-15T10:30:00Z"})
				})
			})
		})
	})
	Method("PostAPIWorkspacesBySlugDraftIssues", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_draft_issues")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/draft-issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.workspace.draft.WorkspaceDraftIssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("slug", String)
			Required("slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:draft-work-items")
			POST("/api/workspaces/{slug}/draft-issues/")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersDraftDraftissuecreateserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(201, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersDraftissuecreateresponseResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugDraftIssuesByPk", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_draft_issues_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/draft-issues/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.workspace.draft.WorkspaceDraftIssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("pk", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:draft-work-items")
			DELETE("/api/workspaces/{slug}/draft-issues/{pk}/")
			Param("pk")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("PatchAPIWorkspacesBySlugDraftIssuesByPk", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "patch_api_workspaces_by_slug_draft_issues_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/workspaces/{slug}/draft-issues/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"partial_update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.workspace.draft.WorkspaceDraftIssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("pk", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:draft-work-items")
			PATCH("/api/workspaces/{slug}/draft-issues/{pk}/")
			Param("pk")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAppSerializersDraftDraftissuecreateserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("PostAPIWorkspacesBySlugDraftToIssueByDraftID", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_draft_to_issue_by_draft_id")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/draft-to-issue/{draft_id}/\"")
		Meta("openapi:extension:x-plane-handler", "\"create_draft_to_issue\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.workspace.draft.WorkspaceDraftIssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("draft_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("draft_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:workspaces")
			POST("/api/workspaces/{slug}/draft-to-issue/{draft_id}/")
			Param("draft_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersIssueIssuecreateserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(201, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssuecreateserializerResponse)
			})
		})
	})
	Method("GetAPIWorkspacesBySlugIssues", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_issues")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"list\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.view.base.WorkspaceViewIssuesViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("slug", String)
			Required("slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			GET("/api/workspaces/{slug}/issues/")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersWorkspaceviewissuespageResponse)
			})
		})
	})
	Method("GetAPIWorkspacesBySlugProjectsByProjectIDArchivedIssues", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_archived_issues")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/projects/{project_id}/archived-issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"list\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.archive.IssueArchiveViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:projects")
			GET("/api/workspaces/{slug}/projects/{project_id}/archived-issues/")
			Param("project_id")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDBulkArchiveIssues", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_bulk_archive_issues")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/bulk-archive-issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.archive.BulkArchiveIssuesEndpoint\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:projects")
			POST("/api/workspaces/{slug}/projects/{project_id}/bulk-archive-issues/")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedRestFrameworkSerializersIssueidsrequestRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersArchiveresponseResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDBulkDeleteIssues", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_bulk_delete_issues")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/bulk-delete-issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"delete\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.base.BulkDeleteIssuesEndpoint\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(Any, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Meta("openapi:tag:projects")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/bulk-delete-issues/")
			Param("project_id")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(bulkDeleteIssueResponse)
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssues", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_cycle_issues")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.cycle.issue.CycleIssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("cycle_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("cycle_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:cycles")
			POST("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/")
			Param("cycle_id")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersCycleCycleissueserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(201, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersCycleCycleissueserializerResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesByIssueID", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_cycle_issues_by_issue_id")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/{issue_id}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.cycle.issue.CycleIssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("cycle_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("issue_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("cycle_id", "issue_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:cycles")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/{issue_id}/")
			Param("cycle_id")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("PatchAPIWorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesByIssueID", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "patch_api_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_cycle_issues_by_issue_id")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/{issue_id}/\"")
		Meta("openapi:extension:x-plane-handler", "\"partial_update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.cycle.issue.CycleIssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("cycle_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("issue_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("cycle_id", "issue_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:cycles")
			PATCH("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/{issue_id}/")
			Param("cycle_id")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAppSerializersCycleCycleissueserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersCycleCycleissueserializerResponse)
			})
		})
	})
	Method("PutAPIWorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesByIssueID", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "put_api_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_cycle_issues_by_issue_id")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PUT /api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/{issue_id}/\"")
		Meta("openapi:extension:x-plane-handler", "\"update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.cycle.issue.CycleIssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("cycle_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("issue_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("cycle_id", "issue_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:cycles")
			PUT("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/{issue_id}/")
			Param("cycle_id")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersCycleCycleissueserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersCycleCycleissueserializerResponse)
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDCyclesByCycleIDTransferIssues", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_transfer_issues")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/transfer-issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.cycle.base.TransferCycleIssueEndpoint\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("cycle_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("cycle_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:cycles")
			POST("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/transfer-issues/")
			Param("cycle_id")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedRestFrameworkSerializersTransfercycleissuesrequestRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersMessageresponseResponse)
			})
		})
	})
	Method("GetAPIWorkspacesBySlugProjectsByProjectIDDeletedIssues", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_deleted_issues")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/projects/{project_id}/deleted-issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.base.DeletedIssuesListViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("search", String, func() {
				Description("A search term.")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("slug", String)
			Required("project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:projects")
			GET("/api/workspaces/{slug}/projects/{project_id}/deleted-issues/")
			Param("project_id")
			Param("search")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDInboxIssues", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_inbox_issues")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/inbox-issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.intake.base.IntakeIssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:intake")
			POST("/api/workspaces/{slug}/projects/{project_id}/inbox-issues/")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersIntakeIntakeissueserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIntakeIntakeissueserializerResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDInboxIssuesByPk", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_inbox_issues_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/inbox-issues/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.intake.base.IntakeIssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:intake")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/inbox-issues/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("PatchAPIWorkspacesBySlugProjectsByProjectIDInboxIssuesByPk", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "patch_api_workspaces_by_slug_projects_by_project_id_inbox_issues_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/workspaces/{slug}/projects/{project_id}/inbox-issues/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"partial_update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.intake.base.IntakeIssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:intake")
			PATCH("/api/workspaces/{slug}/projects/{project_id}/inbox-issues/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAppSerializersIntakeIntakeissueserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIntakeIntakeissueserializerResponse)
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDIntakeIssues", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_intake_issues")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/intake-issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.intake.base.IntakeIssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:intake")
			POST("/api/workspaces/{slug}/projects/{project_id}/intake-issues/")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersIntakeIntakeissueserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIntakeIntakeissueserializerResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDIntakeIssuesByPk", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_intake_issues_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/intake-issues/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.intake.base.IntakeIssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:intake")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/intake-issues/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("PatchAPIWorkspacesBySlugProjectsByProjectIDIntakeIssuesByPk", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "patch_api_workspaces_by_slug_projects_by_project_id_intake_issues_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/workspaces/{slug}/projects/{project_id}/intake-issues/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"partial_update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.intake.base.IntakeIssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:intake")
			PATCH("/api/workspaces/{slug}/projects/{project_id}/intake-issues/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAppSerializersIntakeIntakeissueserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIntakeIntakeissueserializerResponse)
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDIssueDates", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_issue_dates")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/issue-dates/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.base.IssueBulkUpdateDateEndpoint\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:projects")
			POST("/api/workspaces/{slug}/projects/{project_id}/issue-dates/")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedRestFrameworkSerializersIssuedateupdatesrequestRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersMessageresponseResponse)
			})
		})
	})
	Method("GetAPIWorkspacesBySlugProjectsByProjectIDIssuesDetail", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_issues_detail")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/projects/{project_id}/issues-detail/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.base.IssueDetailEndpoint\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:projects")
			GET("/api/workspaces/{slug}/projects/{project_id}/issues-detail/")
			Param("project_id")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
			})
		})
	})
	Method("GetAPIWorkspacesBySlugProjectsByProjectIDIssues", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_issues")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/projects/{project_id}/issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"list\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.base.IssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			GET("/api/workspaces/{slug}/projects/{project_id}/issues/")
			Param("project_id")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDIssues", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_issues")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.base.IssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			POST("/api/workspaces/{slug}/projects/{project_id}/issues/")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersIssueIssuecreateserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(201, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssuecreateserializerResponse)
			})
		})
	})
	Method("GetAPIWorkspacesBySlugProjectsByProjectIDIssuesList", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_issues_list")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/projects/{project_id}/issues/list/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.base.IssueListEndpoint\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			GET("/api/workspaces/{slug}/projects/{project_id}/issues/list/")
			Param("project_id")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
			})
		})
	})
	Method("GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueRelation", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_issue_relation")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-relation/\"")
		Meta("openapi:extension:x-plane-handler", "\"list\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.relation.IssueRelationViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("search", String, func() {
				Description("A search term.")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("slug", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			GET("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-relation/")
			Param("issue_id")
			Param("project_id")
			Param("search")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueRelation", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_issue_relation")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-relation/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.relation.IssueRelationViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			POST("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-relation/")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersIssueIssuerelationserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(201, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssuerelationserializerResponse)
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDModules", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_modules")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/modules/\"")
		Meta("openapi:extension:x-plane-handler", "\"create_issue_modules\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.module.issue.ModuleIssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:modules")
			POST("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/modules/")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedRestFrameworkSerializersIssuemodulesrequestRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(201, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersMessageresponseResponse)
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDRemoveRelation", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_remove_relation")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/remove-relation/\"")
		Meta("openapi:extension:x-plane-handler", "\"remove_relation\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.relation.IssueRelationViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			POST("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/remove-relation/")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersIssueIssuerelationserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(204, func() {
				ContentType("application/json")
			})
		})
	})
	Method("GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDSubIssues", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_sub_issues")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/sub-issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.sub_issue.SubIssuesEndpoint\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("search", String, func() {
				Description("A search term.")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("slug", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			GET("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/sub-issues/")
			Param("issue_id")
			Param("project_id")
			Param("search")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDSubIssues", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_sub_issues")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/sub-issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.sub_issue.SubIssuesEndpoint\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			POST("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/sub-issues/")
			Param("issue_id")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedRestFrameworkSerializersSubissueidsrequestRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersSubissuesresponseResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByPk", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_issues_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/issues/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.base.IssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/issues/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByPk", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_issues_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/projects/{project_id}/issues/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"retrieve\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.base.IssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			GET("/api/workspaces/{slug}/projects/{project_id}/issues/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssueserializerResponse)
			})
		})
	})
	Method("PatchAPIWorkspacesBySlugProjectsByProjectIDIssuesByPk", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "patch_api_workspaces_by_slug_projects_by_project_id_issues_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/workspaces/{slug}/projects/{project_id}/issues/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"partial_update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.base.IssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			PATCH("/api/workspaces/{slug}/projects/{project_id}/issues/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAppSerializersIssueIssuecreateserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(204, func() {
				ContentType("application/json")
			})
		})
	})
	Method("PutAPIWorkspacesBySlugProjectsByProjectIDIssuesByPk", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "put_api_workspaces_by_slug_projects_by_project_id_issues_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PUT /api/workspaces/{slug}/projects/{project_id}/issues/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.base.IssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			PUT("/api/workspaces/{slug}/projects/{project_id}/issues/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersIssueIssuecreateserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssuecreateserializerResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByPkArchive", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_issues_by_pk_archive")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/issues/{pk}/archive/\"")
		Meta("openapi:extension:x-plane-handler", "\"unarchive\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.archive.IssueArchiveViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/issues/{pk}/archive/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByPkArchive", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_issues_by_pk_archive")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/projects/{project_id}/issues/{pk}/archive/\"")
		Meta("openapi:extension:x-plane-handler", "\"retrieve\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.archive.IssueArchiveViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			GET("/api/workspaces/{slug}/projects/{project_id}/issues/{pk}/archive/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssueflatserializerResponse)
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByPkArchive", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_issues_by_pk_archive")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/issues/{pk}/archive/\"")
		Meta("openapi:extension:x-plane-handler", "\"archive\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.archive.IssueArchiveViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			POST("/api/workspaces/{slug}/projects/{project_id}/issues/{pk}/archive/")
			Param("pk")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersIssueIssueflatserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssueflatserializerResponse)
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDIssues", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_modules_by_module_id_issues")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"create_module_issues\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.module.issue.ModuleIssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("module_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("module_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			POST("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/")
			Param("module_id")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedRestFrameworkSerializersModuleissuesrequestRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(201, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersMessageresponseResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDIssuesByIssueID", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_modules_by_module_id_issues_by_issue_id")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/{issue_id}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.module.issue.ModuleIssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("module_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("issue_id", "module_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/{issue_id}/")
			Param("issue_id")
			Param("module_id")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("PatchAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDIssuesByIssueID", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "patch_api_workspaces_by_slug_projects_by_project_id_modules_by_module_id_issues_by_issue_id")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/{issue_id}/\"")
		Meta("openapi:extension:x-plane-handler", "\"partial_update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.module.issue.ModuleIssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("module_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("issue_id", "module_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			PATCH("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/{issue_id}/")
			Param("issue_id")
			Param("module_id")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAppSerializersModuleModuleissueserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersModuleModuleissueserializerResponse)
			})
		})
	})
	Method("PutAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDIssuesByIssueID", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "put_api_workspaces_by_slug_projects_by_project_id_modules_by_module_id_issues_by_issue_id")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PUT /api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/{issue_id}/\"")
		Meta("openapi:extension:x-plane-handler", "\"update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.module.issue.ModuleIssueViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("issue_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("module_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("issue_id", "module_id", "project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			PUT("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/{issue_id}/")
			Param("issue_id")
			Param("module_id")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersModuleModuleissueserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersModuleModuleissueserializerResponse)
			})
		})
	})
	Method("GetAPIWorkspacesBySlugProjectsByProjectIDSearchIssues", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_search_issues")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/projects/{project_id}/search-issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.search.issue.IssueSearchEndpoint\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("type_id", String, func() { Format(FormatUUID) })
			Attribute("child_type_id", String, func() { Format(FormatUUID) })
			Attribute("search", String, func() {
				Description("A search term.")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("slug", String)
			Required("project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:projects")
			GET("/api/workspaces/{slug}/projects/{project_id}/search-issues/")
			Param("project_id")
			Param("type_id")
			Param("child_type_id")
			Param("search")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
			})
		})
	})
	Method("GetAPIWorkspacesBySlugProjectsByProjectIDV2Issues", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_v2_issues")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/projects/{project_id}/v2/issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"list\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.base.IssuePaginatedViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("search", String, func() {
				Description("A search term.")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("slug", String)
			Required("project_id", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			GET("/api/workspaces/{slug}/projects/{project_id}/v2/issues/")
			Param("project_id")
			Param("search")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersIssueglobalpageResponse)
			})
		})
	})
	Method("GetAPIWorkspacesBySlugUserIssuesByUserID", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_user_issues_by_user_id")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/user-issues/{user_id}/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.workspace.WorkspaceViewerPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.workspace.user.WorkspaceUserProfileIssuesEndpoint\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("slug", String)
			Attribute("user_id", String, func() {
				Format(FormatUUID)
			})
			Required("slug", "user_id")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:workspaces")
			GET("/api/workspaces/{slug}/user-issues/{user_id}/")
			Param("slug")
			Param("user_id")
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersWorkspaceuserissuespageResponse)
			})
		})
	})
	Method("GetAPIWorkspacesBySlugWorkItemsProjectIdentifierIssueIdentifier", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_work_items__project_identifier___issue_identifier")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/work-items/{project_identifier}-{issue_identifier}/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.base.IssueDetailIdentifierEndpoint\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("identifier", String, func() {
				Description("Project identifier and issue sequence in one segment; Django lookup preserves the complete spelling.")
				Example("IC-1")
			})
			Attribute("slug", String)
			Required("identifier", "slug")
		})
		Result(workItemResponse)
		workItemErrors()
		HTTP(func() {
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:work-items")
			GET("/api/workspaces/{slug}/work-items/{identifier}/")
			Param("identifier")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersIssueIssuedetailserializerResponse)
			})
		})
	})
})

// The source payloads accept JSON, form and multipart bodies. Their imported
// raw request streams preserve parser precedence and Django field coercions.
var PlaneSession = SessionAuth("PlaneSession", func() {
	CookieTransport(ImportedPlaneSessionCookieSecurity, "", func() { CookieName("session-id") })
})
var workItemResponse = Type("WorkItemResponse", func() {})

// Django errors include field-keyed objects, nested relation errors, and scalar
// values. Project the complete JSON body without changing its shape or precision.
var workItemError = Type("WorkItemError", func() {
	ErrorName("kind", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Required("kind", "body")
})

func workItemErrors() {
	for _, item := range []struct {
		name string
		code int
	}{
		{"status_400", 400}, {"status_401", 401}, {"status_403", 403},
		{"status_404", 404}, {"status_405", 405}, {"status_406", 406},
		{"status_409", 409}, {"status_413", 413}, {"status_415", 415}, {"status_429", 429}, {"status_500", 500},
	} {
		Error(item.name, workItemError, func() {
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
