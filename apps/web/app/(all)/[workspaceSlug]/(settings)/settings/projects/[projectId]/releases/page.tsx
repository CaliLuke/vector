import { createFileRoute } from "@tanstack/react-router";
import { Package } from "lucide-react";
import { Breadcrumbs } from "@plane/ui";
import { PageHead } from "@/components/core/page-title";
import { SettingsContentWrapper } from "@/components/settings/content-wrapper";
import { SettingsHeading } from "@/components/settings/heading";
import { SettingsPageHeader } from "@/components/settings/page-header";
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { ProjectReleases } from "@/components/releases/project-releases";

function ProjectReleasesPage() {
  const { workspaceSlug, projectId } = Route.useParams();
  return (
    <SettingsContentWrapper
      header={
        <SettingsPageHeader
          leftItem={
            <Breadcrumbs>
              <Breadcrumbs.Item
                component={<BreadcrumbLink label="Releases" icon={<Package className="size-4 text-tertiary" />} />}
              />
            </Breadcrumbs>
          }
        />
      }
    >
      <PageHead title="Releases" />
      <SettingsHeading
        title="Releases"
        description="Create and manage the releases available in this project’s work item properties."
      />
      <div className="mt-6">
        <ProjectReleases key={`${workspaceSlug}:${projectId}`} workspaceSlug={workspaceSlug} projectId={projectId} />
      </div>
    </SettingsContentWrapper>
  );
}
export const Route = createFileRoute("/_authenticated/$workspaceSlug/settings/projects/$projectId/releases")({
  component: ProjectReleasesPage,
});
