/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import type { TIssueServiceType, TWorkItemWidgets } from "@plane/types";
// hooks
import { useWorkItemDetail, useWorkItemSubItems } from "@/hooks/use-work-item-detail";
import { useIssueLinks, useIssueRelations } from "@/hooks/use-issue-resources";
import { countVisibleRelations } from "./relations/count";
import { useAttachmentOperations } from "./attachments/helper";
import { useTimeLineRelationOptions } from "@/components/relations";
// local imports
import { AttachmentsCollapsible } from "./attachments";
import { LinksCollapsible } from "./links";
import { RelationsCollapsible } from "./relations";
import { SubIssuesCollapsible } from "./sub-issues";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  disabled: boolean;
  issueServiceType: TIssueServiceType;
  hideWidgets?: TWorkItemWidgets[];
};

function getWidgetVisibility({
  subIssueCount,
  relationCount,
  relationError,
  linkCount,
  linkError,
  attachmentCount,
  uploadCount,
  attachmentError,
  hiddenWidgets,
}: {
  subIssueCount: number;
  relationCount: number;
  relationError: boolean;
  linkCount: number;
  linkError: boolean;
  attachmentCount: number;
  uploadCount: number;
  attachmentError: boolean;
  hiddenWidgets: TWorkItemWidgets[] | undefined;
}) {
  const isVisible = (widget: TWorkItemWidgets) => !hiddenWidgets?.includes(widget);
  return {
    subIssues: subIssueCount > 0 && isVisible("sub-work-items"),
    relations: (relationCount > 0 || relationError) && isVisible("relations"),
    links: (linkCount > 0 || linkError) && isVisible("links"),
    attachments: attachmentCount > 0 || ((uploadCount > 0 || attachmentError) && isVisible("attachments")),
  };
}

function hasResourceError(denied: boolean, queryError: boolean, accessError = false) {
  return !denied && (queryError || accessError);
}

export function IssueDetailWidgetCollapsibles(props: Props) {
  const { workspaceSlug, projectId, issueId, disabled, issueServiceType, hideWidgets } = props;
  const { issue } = useWorkItemDetail(workspaceSlug, projectId, issueId, issueServiceType);
  const subItems = useWorkItemSubItems(workspaceSlug, projectId, issueId, issueServiceType);
  const subIssues = subItems.subItems;
  const ISSUE_RELATION_OPTIONS = useTimeLineRelationOptions();
  const attachmentHelpers = useAttachmentOperations(workspaceSlug, projectId, issueId, issueServiceType);
  const {
    relations,
    query: relationsQuery,
    isError: relationsAccessError,
    isDenied: relationsDenied,
  } = useIssueRelations({
    workspaceSlug,
    projectId,
    issueId,
    serviceType: issueServiceType,
  });
  const {
    links,
    query: linksQuery,
    isError: linksAccessError,
    isDenied: linksDenied,
  } = useIssueLinks({
    workspaceSlug,
    projectId,
    issueId,
    serviceType: issueServiceType,
  });
  const issueRelationsCount = countVisibleRelations(relations, ISSUE_RELATION_OPTIONS);
  const attachmentUploads = attachmentHelpers.snapshot.uploadStatus;
  const attachmentsCount = attachmentHelpers.snapshot.attachments?.length ?? issue?.attachment_count ?? 0;
  const visibleWidgets = getWidgetVisibility({
    subIssueCount: subIssues?.length ?? 0,
    relationCount: issueRelationsCount,
    relationError: hasResourceError(relationsDenied, relationsQuery.isError, relationsAccessError),
    linkCount: linksQuery.isSuccess ? (links?.length ?? 0) : (issue?.link_count ?? 0),
    linkError: hasResourceError(linksDenied, linksQuery.isError, linksAccessError),
    attachmentCount: attachmentsCount,
    uploadCount: attachmentUploads?.length ?? 0,
    attachmentError: hasResourceError(attachmentHelpers.snapshot.isDenied, attachmentHelpers.snapshot.isError),
    hiddenWidgets: hideWidgets,
  });

  return (
    <div className="flex flex-col">
      {visibleWidgets.subIssues && (
        <SubIssuesCollapsible
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={issueId}
          disabled={disabled}
          issueServiceType={issueServiceType}
        />
      )}
      {visibleWidgets.relations && (
        <RelationsCollapsible
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={issueId}
          disabled={disabled}
          issueServiceType={issueServiceType}
        />
      )}
      {visibleWidgets.links && (
        <LinksCollapsible
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={issueId}
          disabled={disabled}
          issueServiceType={issueServiceType}
        />
      )}
      {visibleWidgets.attachments && (
        <AttachmentsCollapsible
          workspaceSlug={workspaceSlug}
          issueId={issueId}
          disabled={disabled}
          issueServiceType={issueServiceType}
          attachmentHelpers={attachmentHelpers}
        />
      )}
    </div>
  );
}
