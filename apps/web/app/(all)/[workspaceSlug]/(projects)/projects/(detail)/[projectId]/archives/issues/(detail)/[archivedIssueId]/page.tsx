/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute, useRouter } from "@tanstack/react-router";

// ui
import { Banner } from "@plane/propel/banner";
import { Button } from "@plane/propel/button";
import { ArchiveIcon } from "@plane/propel/icons";
import { Loader } from "@plane/ui";
// components
import { PageHead } from "@/components/core/page-title";
import { IssueDetailRoot } from "@/components/issues/issue-detail";
// constants
// hooks
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useWorkItemDetail } from "@/hooks/use-work-item-detail";

function ArchivedIssueDetailsPage(): React.JSX.Element | null {
  // router
  const { workspaceSlug, projectId, archivedIssueId } = Route.useParams();
  const router = useRouter();
  // states
  // hooks
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  const detail = useWorkItemDetail(workspaceSlug, projectId, archivedIssueId);

  // derived values
  const issue = detail.issue;
  const project = issue ? projectRows?.find((projectRow) => projectRow.id === issue.project_id) : undefined;
  const pageTitle = project && issue ? `${project.identifier}-${issue.sequence_id} ${issue.name}` : undefined;

  if (!issue) return <></>;

  const issueLoader = detail.isPending;

  return (
    <>
      <PageHead title={pageTitle} />
      {issueLoader ? (
        <Loader className="flex h-full gap-5 p-5">
          <div className="basis-2/3 space-y-2">
            <Loader.Item height="30px" width="40%" />
            <Loader.Item height="15px" width="60%" />
            <Loader.Item height="15px" width="60%" />
            <Loader.Item height="15px" width="40%" />
          </div>
          <div className="basis-1/3 space-y-3">
            <Loader.Item height="30px" />
            <Loader.Item height="30px" />
            <Loader.Item height="30px" />
            <Loader.Item height="30px" />
          </div>
        </Loader>
      ) : (
        <>
          <Banner
            variant="warning"
            title="This work item has been archived. Visit the Archives section to restore it."
            icon={<ArchiveIcon className="size-4" />}
            action={
              <Button
                variant="secondary"
                onClick={() =>
                  void router.navigate({ href: `/${workspaceSlug}/projects/${projectId}/archives/issues/` })
                }
              >
                Go to archives
              </Button>
            }
            className="border-b border-subtle"
          />
          <div className="flex h-full overflow-hidden">
            <div className="h-full w-full space-y-3 divide-y-2 divide-subtle-1 overflow-y-auto">
              <IssueDetailRoot
                workspaceSlug={workspaceSlug}
                projectId={projectId}
                issueId={archivedIssueId}
                is_archived
              />
            </div>
          </div>
        </>
      )}
    </>
  );
}

export const Route = createFileRoute(
  "/_authenticated/$workspaceSlug/_projects/projects/$projectId/archives/issues/$archivedIssueId/"
)({
  component: ArchivedIssueDetailsPage,
});
