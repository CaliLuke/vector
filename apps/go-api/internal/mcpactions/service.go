package mcpactions

import (
	"context"
	"errors"
)

// Applications supplies the shared application operations behind generated tools.
// Read repositories must use the primary database for fresh scope checks.
type Applications struct {
	Customers       customerApplication
	Templates       templateApplication
	Features        featureApplication
	Initiatives     initiativeApplication
	Pages           pageApplication
	Releases        releaseApplication
	Identity        identityReader
	ProjectMembers  projectMemberReader
	MemberDirectory memberDirectory
	Projects        projectReader
	ProjectWrites   projectWriter
	Cycles          projectResourceReader
	CycleWrites     resourceWriter
	Modules         projectResourceReader
	ModuleWrites    resourceWriter
	Labels          projectResourceReader
	LabelWrites     labelWriter
	Collaboration   collaborationReader
	Activities      activityReader
	Milestones      milestoneApplication
	States          stateApplication
	Estimates       estimateApplication
	Types           typeApplication
	Attachments     attachmentApplication
	WorkLogs        workLogApplication
	Relations       relationApplication
	Properties      propertyApplication
}
type Service struct {
	customers          customerApplication
	templates          templateApplication
	features           featureApplication
	initiatives        initiativeApplication
	pages              pageApplication
	releases           releaseApplication
	logContext         context.Context
	identity           identityReader
	projects           projectMemberReader
	directory          memberDirectory
	projectReads       projectReader
	projectWrites      projectWriter
	cycleReads         projectResourceReader
	cycleWrites        resourceWriter
	moduleReads        projectResourceReader
	moduleWrites       resourceWriter
	labelReads         projectResourceReader
	labelWrites        labelWriter
	collaborationReads collaborationReader
	activityReads      activityReader
	milestones         milestoneApplication
	states             stateApplication
	estimates          estimateApplication
	types              typeApplication
	attachments        attachmentApplication
	workLogs           workLogApplication
	relations          relationApplication
	properties         propertyApplication
	origin             string
}

func New(apps Applications, options ...Option) (*Service, error) {
	for _, dependency := range []any{apps.Customers, apps.Templates, apps.Features, apps.Initiatives, apps.Pages, apps.Releases, apps.Identity, apps.ProjectMembers, apps.MemberDirectory, apps.Projects, apps.ProjectWrites, apps.Cycles, apps.CycleWrites, apps.Modules, apps.ModuleWrites, apps.Labels, apps.LabelWrites, apps.Collaboration, apps.Activities, apps.Milestones, apps.States, apps.Estimates, apps.Types, apps.Attachments, apps.WorkLogs, apps.Relations, apps.Properties} {
		if dependency == nil {
			return nil, errors.New("MCP applications are required")
		}
	}

	service := &Service{customers: apps.Customers, templates: apps.Templates, features: apps.Features, initiatives: apps.Initiatives, pages: apps.Pages, releases: apps.Releases, identity: apps.Identity, projects: apps.ProjectMembers, directory: apps.MemberDirectory, projectReads: apps.Projects, projectWrites: apps.ProjectWrites, cycleReads: apps.Cycles, cycleWrites: apps.CycleWrites, moduleReads: apps.Modules, moduleWrites: apps.ModuleWrites, labelReads: apps.Labels, labelWrites: apps.LabelWrites, collaborationReads: apps.Collaboration, activityReads: apps.Activities, milestones: apps.Milestones, states: apps.States, estimates: apps.Estimates, types: apps.Types, attachments: apps.Attachments, workLogs: apps.WorkLogs, relations: apps.Relations, properties: apps.Properties}
	for _, option := range options {
		option(service)
	}
	return service, nil
}

// WithOrigin supplies the configured application origin for durable effects.
func WithOrigin(origin string) Option { return func(s *Service) { s.origin = origin } }
