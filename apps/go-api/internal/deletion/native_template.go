package deletion

// Native catalogues use the same parent cascades and retention schedule as the
// compatibility models. Audit events deliberately remain outside soft deletion.
func nativeTemplateModel() Model {
	return Model{
		App: "db", Name: "resourcetemplate", Table: "resource_templates", PK: "id", PKType: "uuid", SoftDelete: true, UpdatedAt: true,
		Relations: []Relation{
			{Column: "workspace_id", TargetTable: "workspaces", TargetColumn: "id", OnDelete: "CASCADE", SoftReverse: true},
			{Column: "project_id", TargetTable: "projects", TargetColumn: "id", OnDelete: "CASCADE", SoftReverse: true},
			{Column: "created_by_id", TargetTable: "users", TargetColumn: "id", OnDelete: "SET_NULL", SoftReverse: true},
			{Column: "updated_by_id", TargetTable: "users", TargetColumn: "id", OnDelete: "SET_NULL", SoftReverse: true},
		},
	}
}
