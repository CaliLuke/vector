/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link, useLocation } from "@tanstack/react-router";
import type { MouseEvent } from "react";

// plane imports
import { PriorityIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import { Row, Avatar } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { renderFormattedDate } from "@plane/utils/datetime";
import { getFileURL } from "@plane/utils/file";
import type { IIssueLabel, IUserLite, TInboxIssue } from "@plane/types";
import { EInboxIssueStatus } from "@plane/types";
// components
// hooks
import { useProjectLabels } from "@/hooks/use-labels";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { useIntakePreferences } from "@/providers/intake-preferences";
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { InboxIssueStatus } from "../inbox-issue-status";

function InboxIssueLabels({ labelIds, labels }: { labelIds: string[] | undefined; labels: IIssueLabel[] | undefined }) {
  if (labelIds && labelIds.length > 3) {
    return (
      <div className="relative flex !h-[17.5px] items-center gap-1 rounded-sm border border-strong px-1 text-11">
        <span className="bg-orange-400 h-2 w-2 rounded-full" />
        <span className="max-w-28 truncate normal-case">{`${labelIds.length} labels`}</span>
      </div>
    );
  }

  return (
    <>
      {(labelIds ?? []).map((labelId) => {
        const labelDetails = labels?.find((label) => label.id === labelId);
        if (!labelDetails) return null;
        return (
          <div
            key={labelId}
            className="relative flex !h-[17.5px] items-center gap-1 rounded-sm border border-strong px-1 text-11"
          >
            <span className="h-2 w-2 rounded-full" style={{ backgroundColor: labelDetails.color }} />
            <span className="max-w-28 truncate normal-case">{labelDetails.name}</span>
          </div>
        );
      })}
    </>
  );
}

function InboxIssueCreator({ member }: { member: IUserLite | undefined }) {
  if (!member) return null;
  if (member.email?.includes("intake@plane.so")) {
    return <Avatar src={getFileURL("")} name="Vector" size="md" showTooltip />;
  }
  return <Avatar src={getFileURL(member.avatar_url ?? "")} name={member.display_name} size="md" showTooltip />;
}

type InboxIssueListItemProps = {
  workspaceSlug: string;
  projectId: string;
  projectIdentifier?: string;
  inboxIssue: TInboxIssue;
  setIsMobileSidebar: (value: boolean) => void;
};

export const InboxIssueListItem = function InboxIssueListItem(props: InboxIssueListItemProps) {
  const { workspaceSlug, projectId, inboxIssue, projectIdentifier, setIsMobileSidebar } = props;
  // router
  const searchParams = new URLSearchParams(useLocation({ select: (location) => location.searchStr }));
  const selectedInboxIssueId = searchParams.get("inboxIssueId");
  // store
  const { tab: currentTab } = useIntakePreferences();
  const { labels: projectLabels } = useProjectLabels(workspaceSlug, projectId);
  const { isMobile } = usePlatformOS();
  const { memberById } = useWorkspaceMemberRoster(workspaceSlug);
  const issue = inboxIssue?.issue;

  const handleIssueRedirection = (event: MouseEvent, currentIssueId: string | undefined) => {
    if (selectedInboxIssueId === currentIssueId) event.preventDefault();
    setIsMobileSidebar(false);
  };

  if (!issue) return <></>;

  const createdByDetails = issue?.created_by ? memberById(issue?.created_by)?.member : undefined;

  return (
    <>
      <Link
        id={`inbox-issue-list-item-${issue.id}`}
        key={`${projectId}_${issue.id}`}
        to="/$workspaceSlug/projects/$projectId/intake"
        params={{ workspaceSlug, projectId }}
        search={{ currentTab, inboxIssueId: issue.id }}
        onClick={(e) => handleIssueRedirection(e, issue.id)}
      >
        <Row
          className={cn(
            `relative flex cursor-pointer flex-col gap-2 border border-t-transparent border-r-transparent border-b-subtle-1 border-l-transparent py-4 transition-all hover:bg-accent-primary/5`,
            { "border border-accent-strong": selectedInboxIssueId === issue.id }
          )}
        >
          <div className="space-y-1">
            <div className="relative flex items-center justify-between gap-2">
              <div className="flex-shrink-0 text-11 font-medium text-tertiary">
                {projectIdentifier}-{issue.sequence_id}
              </div>
              <div className="flex items-center gap-2">
                {inboxIssue.status !== EInboxIssueStatus.PENDING && (
                  <InboxIssueStatus inboxIssue={inboxIssue} iconSize={12} />
                )}
              </div>
            </div>
            <h3 className="w-full truncate text-13">{issue.name}</h3>
          </div>

          <div className="flex items-center justify-between">
            <div className="flex flex-wrap items-center gap-2">
              <Tooltip
                tooltipHeading="Created on"
                tooltipContent={`${renderFormattedDate(issue.created_at ?? "")}`}
                isMobile={isMobile}
              >
                <div className="text-11 text-secondary">{renderFormattedDate(issue.created_at ?? "")}</div>
              </Tooltip>

              <div className="rounded-full border-2 border-strong-1" />

              {issue.priority && (
                <Tooltip tooltipHeading="Priority" tooltipContent={issue.priority}>
                  <PriorityIcon priority={issue.priority} withContainer className="h-3 w-3" />
                </Tooltip>
              )}

              <InboxIssueLabels labelIds={issue.label_ids} labels={projectLabels} />
            </div>
            {/* created by */}
            <InboxIssueCreator member={createdByDetails} />
          </div>
        </Row>
      </Link>
    </>
  );
};
