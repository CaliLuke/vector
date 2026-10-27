package mcpactions

import (
	"context"
	"errors"
)

// Applications supplies the shared application operations behind generated tools.
// Read repositories must use the primary database for fresh scope checks.
type Applications struct {
	Identity        identityReader
	ProjectMembers  projectMemberReader
	MemberDirectory memberDirectory
	Projects        projectReader
	ProjectWrites   projectWriter
	Cycles          projectResourceReader
	Modules         projectResourceReader
	Labels          projectResourceReader
}
type Service struct {
	logContext    context.Context
	identity      identityReader
	projects      projectMemberReader
	directory     memberDirectory
	projectReads  projectReader
	projectWrites projectWriter
	cycleReads    projectResourceReader
	moduleReads   projectResourceReader
	labelReads    projectResourceReader
}

func New(apps Applications, options ...Option) (*Service, error) {
	if apps.Identity == nil || apps.ProjectMembers == nil || apps.MemberDirectory == nil || apps.Projects == nil || apps.ProjectWrites == nil || apps.Cycles == nil || apps.Modules == nil || apps.Labels == nil {
		return nil, errors.New("MCP applications are required")
	}
	service := &Service{identity: apps.Identity, projects: apps.ProjectMembers, directory: apps.MemberDirectory, projectReads: apps.Projects, projectWrites: apps.ProjectWrites, cycleReads: apps.Cycles, moduleReads: apps.Modules, labelReads: apps.Labels}
	for _, option := range options {
		option(service)
	}
	return service, nil
}
