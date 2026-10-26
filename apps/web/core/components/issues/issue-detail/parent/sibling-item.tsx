/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TIssue } from "@plane/types";
// ui
import { CustomMenu } from "@plane/ui";
// helpers
import { generateWorkItemLink } from "@plane/utils/work-item";
// hooks
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
// components
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";

type TIssueParentSiblingItem = {
  workspaceSlug: string;
  issue: TIssue;
};

export function IssueParentSiblingItem(props: TIssueParentSiblingItem) {
  const { workspaceSlug, issue: issueDetail } = props;
  // hooks
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;

  const projectDetails =
    (issueDetail.project_id && projectRows?.find((project) => project.id === issueDetail.project_id)) || undefined;

  const workItemLink = generateWorkItemLink({
    workspaceSlug,
    projectId: issueDetail?.project_id,
    issueId: issueDetail?.id,
    projectIdentifier: projectDetails?.identifier,
    sequenceId: issueDetail?.sequence_id,
  });

  return (
    <>
      <CustomMenu.MenuItem
        key={issueDetail.id}
        onClick={() => window.open(workItemLink, "_blank", "noopener,noreferrer")}
      >
        <div className="flex items-center gap-2 py-0.5">
          {issueDetail.project_id && projectDetails?.identifier && (
            <IssueIdentifier
              projectId={issueDetail.project_id}
              issueTypeId={issueDetail.type_id}
              projectIdentifier={projectDetails?.identifier}
              issueSequenceId={issueDetail.sequence_id}
              size="xs"
            />
          )}
        </div>
      </CustomMenu.MenuItem>
    </>
  );
}
