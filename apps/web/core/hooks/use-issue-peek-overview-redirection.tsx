/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// types
import { useParams, useRouter } from "@tanstack/react-router";
import type { TIssue } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
// helpers
import { generateWorkItemLink } from "@plane/utils/work-item";
// hooks
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { useWorkspaceProjectDetails } from "./use-projects";

const useIssuePeekOverviewRedirection = (isEpic: boolean = false) => {
  // router
  const router = useRouter();
  const { workspaceSlug } = useParams({ strict: false });
  const projects = useWorkspaceProjectDetails(workspaceSlug);
  //   store hooks
  const { getIsIssuePeeked, setPeekIssue } = useIssueDetailUiState(
    isEpic ? EIssueServiceType.EPICS : EIssueServiceType.ISSUES
  );

  const handleRedirection = (
    targetWorkspaceSlug: string | undefined,
    issue: TIssue | undefined,
    isMobile = false,
    nestingLevel?: number
  ) => {
    if (!issue) return;
    const { project_id, id, archived_at, tempId } = issue;
    const projectIdentifier = projects.projects?.find((project) => project.id === project_id)?.identifier;

    const workItemLink = generateWorkItemLink({
      workspaceSlug: targetWorkspaceSlug,
      projectId: project_id,
      issueId: id,
      projectIdentifier,
      sequenceId: issue?.sequence_id,
      isEpic,
      isArchived: !!archived_at,
    });
    if (targetWorkspaceSlug && project_id && id && !getIsIssuePeeked(id) && !tempId) {
      if (isMobile) {
        void router.navigate({ href: workItemLink });
      } else {
        setPeekIssue({
          workspaceSlug: targetWorkspaceSlug,
          projectId: project_id,
          issueId: id,
          nestingLevel,
          isArchived: !!archived_at,
        });
      }
    }
  };

  return { handleRedirection };
};

export default useIssuePeekOverviewRedirection;
