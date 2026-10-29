/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TIssueServiceType } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
import { useIssueLinks } from "@/hooks/use-issue-resources";
import { IssueLinkDetail } from "./link-detail";
// hooks
import type { TLinkOperations } from "./root";

type TLinkOperationsModal = Exclude<TLinkOperations, "create">;

export type TIssueLinkList = {
  issueId: string;
  projectId: string;
  workspaceSlug: string;
  issueServiceType?: TIssueServiceType;
  linkOperations: TLinkOperationsModal;
  disabled?: boolean;
};

export function IssueLinkList(props: TIssueLinkList) {
  // props
  const {
    issueId,
    projectId,
    workspaceSlug,
    issueServiceType = EIssueServiceType.ISSUES,
    linkOperations,
    disabled = false,
  } = props;
  const { links: issueLinks, query } = useIssueLinks({
    workspaceSlug,
    projectId,
    issueId,
    serviceType: issueServiceType,
  });

  if (query.isError) {
    return (
      <button type="button" className="text-11 text-danger-primary" onClick={() => void query.refetch()}>
        Retry
      </button>
    );
  }
  if (!issueLinks) return <></>;

  return (
    <div className="space-y-2">
      {issueLinks &&
        issueLinks.length > 0 &&
        issueLinks.map((link) => (
          <IssueLinkDetail
            key={link.id}
            link={link}
            linkOperations={linkOperations}
            isNotAllowed={disabled}
            workspaceSlug={workspaceSlug}
          />
        ))}
    </div>
  );
}
