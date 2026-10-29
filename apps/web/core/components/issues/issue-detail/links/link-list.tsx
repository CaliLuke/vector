/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import type { TIssueServiceType } from "@plane/types";
import { useIssueLinks } from "@/hooks/use-issue-resources";
// local imports
import { IssueLinkItem } from "./link-item";
import { IssueResourceRetryButton } from "../resource-retry-button";
import type { TLinkOperations } from "@/hooks/use-issue-link-operations";

type TLinkOperationsModal = Pick<TLinkOperations, "remove" | "pending">;

type TLinkList = {
  issueId: string;
  workspaceSlug: string;
  projectId: string;
  linkOperations: TLinkOperationsModal;
  disabled?: boolean;
  issueServiceType: TIssueServiceType;
};

export function LinkList(props: TLinkList) {
  // props
  const { issueId, workspaceSlug, projectId, linkOperations, disabled = false, issueServiceType } = props;
  const {
    links: issueLinks,
    query,
    isError,
    retry,
  } = useIssueLinks({
    workspaceSlug,
    projectId,
    issueId,
    serviceType: issueServiceType,
  });

  if (query.isError || isError)
    return <IssueResourceRetryButton onRetry={() => void (isError ? retry() : query.refetch())} />;
  if (!issueLinks) return null;

  return (
    <div className="flex flex-col gap-2 pt-4">
      {issueLinks.map((link) => (
        <IssueLinkItem
          key={link.id}
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={issueId}
          link={link}
          linkOperations={linkOperations}
          isNotAllowed={disabled}
          issueServiceType={issueServiceType}
        />
      ))}
    </div>
  );
}
