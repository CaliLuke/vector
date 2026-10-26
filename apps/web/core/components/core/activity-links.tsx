/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { Tooltip } from "@plane/propel/tooltip";
import type { IIssueActivity } from "@plane/types";
import { generateWorkItemLink } from "@plane/utils/work-item";
import { usePlatformOS } from "@/hooks/use-platform-os";

export function IssueLink({ activity }: { activity: IIssueActivity }) {
  // router params
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const { isMobile } = usePlatformOS();

  const workItemLink = generateWorkItemLink({
    workspaceSlug: workspaceSlug ?? activity.workspace_detail?.slug,
    projectId: activity.project,
    issueId: activity.issue,
    projectIdentifier: activity.project_detail.identifier,
    sequenceId: activity.issue_detail?.sequence_id,
  });

  return (
    <Tooltip
      tooltipContent={activity.issue_detail ? activity.issue_detail.name : "This work item has been deleted"}
      isMobile={isMobile}
    >
      {activity.issue_detail ? (
        <a
          aria-disabled={activity.issue === null}
          href={workItemLink}
          target={activity.issue === null ? "_self" : "_blank"}
          rel={activity.issue === null ? "" : "noopener noreferrer"}
          className="inline items-center gap-1 font-medium text-primary hover:underline"
        >
          <span className="whitespace-nowrap">{`${activity.project_detail.identifier}-${activity.issue_detail.sequence_id}`}</span>{" "}
          <span className="font-regular break-all">{activity.issue_detail.name}</span>
        </a>
      ) : (
        <span className="inline-flex items-center gap-1 font-medium whitespace-nowrap text-primary">
          {" a work item"}{" "}
        </span>
      )}
    </Tooltip>
  );
}

export function UserLink({ activity }: { activity: IIssueActivity }) {
  // router params
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });

  return (
    <a
      href={`/${workspaceSlug ?? activity.workspace_detail?.slug}/profile/${
        activity.new_identifier ?? activity.old_identifier
      }`}
      target="_blank"
      rel="noopener noreferrer"
      className="inline-flex items-center font-medium text-primary hover:underline"
    >
      {activity.new_value && activity.new_value !== "" ? activity.new_value : activity.old_value}
    </a>
  );
}
