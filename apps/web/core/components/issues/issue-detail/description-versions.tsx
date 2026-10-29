/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TIssue } from "@plane/types";
import { DescriptionVersionsRoot } from "@/components/core/description-versions";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { WorkItemVersionService } from "@/services/issue";

const workItemVersionService = new WorkItemVersionService();

type Props = {
  workspaceSlug: string;
  projectId: string;
  issue: TIssue;
  disabled: boolean;
  onRestore: (descriptionHTML: string) => void;
};

export function IssueDescriptionVersions({ workspaceSlug, projectId, issue, disabled, onRestore }: Props) {
  const { memberById } = useWorkspaceMemberRoster(workspaceSlug);
  return (
    <DescriptionVersionsRoot
      className="flex-shrink-0"
      entityInformation={{
        createdAt: new Date(issue.created_at ?? 0),
        createdByDisplayName: memberById(issue.created_by ?? "")?.member.display_name ?? "",
        id: issue.id,
        isRestoreDisabled: disabled,
      }}
      fetchHandlers={{
        listDescriptionVersions: (issueId) =>
          workItemVersionService.listDescriptionVersions(workspaceSlug, projectId, issueId),
        retrieveDescriptionVersion: (issueId, versionId) =>
          workItemVersionService.retrieveDescriptionVersion(workspaceSlug, projectId, issueId, versionId),
      }}
      handleRestore={onRestore}
      projectId={projectId}
      workspaceSlug={workspaceSlug}
    />
  );
}
