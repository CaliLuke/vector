import { createFileRoute } from "@tanstack/react-router";
import { Shapes } from "lucide-react";
import { Breadcrumbs } from "@plane/ui";
import { PageHead } from "@/components/core/page-title";
import { SettingsContentWrapper } from "@/components/settings/content-wrapper";
import { SettingsHeading } from "@/components/settings/heading";
import { SettingsPageHeader } from "@/components/settings/page-header";
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { ProjectTypes } from "@/components/work-item-types/project-types";

function ProjectTypesPage() {
  const { workspaceSlug, projectId } = Route.useParams();
  return (
    <SettingsContentWrapper
      header={
        <SettingsPageHeader
          leftItem={
            <Breadcrumbs>
              <Breadcrumbs.Item
                component={
                  <BreadcrumbLink label="Work item types" icon={<Shapes className="size-4 text-tertiary" />} />
                }
              />
            </Breadcrumbs>
          }
        />
      }
    >
      <PageHead title="Work item types" />
      <SettingsHeading
        title="Work item types"
        description="Choose which workspace types are available for new work in this project."
      />
      <div className="mt-6">
        <ProjectTypes key={`${workspaceSlug}:${projectId}`} workspaceSlug={workspaceSlug} projectId={projectId} />
      </div>
    </SettingsContentWrapper>
  );
}
export const Route = createFileRoute("/_authenticated/$workspaceSlug/settings/projects/$projectId/work-item-types")({
  component: ProjectTypesPage,
});
