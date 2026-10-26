/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// icons
import { MembersPropertyIcon } from "@plane/propel/icons";
// hooks;
import type { TIssueActivity } from "@plane/types";
// components
import { IssueActivityBlockComponent, IssueLink } from "./";

type TIssueAssigneeActivity = { activity: TIssueActivity; showIssue?: boolean; ends: "top" | "bottom" | undefined };

export const IssueAssigneeActivity = function IssueAssigneeActivity(props: TIssueAssigneeActivity) {
  const { activity, ends, showIssue = true } = props;
  // hooks

  if (!activity) return <></>;
  return (
    <IssueActivityBlockComponent
      icon={<MembersPropertyIcon className="h-3.5 w-3.5 flex-shrink-0 text-secondary" />}
      activity={activity}
      ends={ends}
    >
      <>
        {activity.old_value === "" ? `added a new assignee ` : `removed the assignee `}
        <a
          href={`/${activity.workspace_detail?.slug}/profile/${activity.new_identifier ?? activity.old_identifier}`}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center font-medium text-primary capitalize hover:underline"
        >
          {activity.new_value && activity.new_value !== "" ? activity.new_value : activity.old_value}
        </a>
        {showIssue && (activity.old_value === "" ? ` to ` : ` from `)}
        {showIssue && <IssueLink activity={activity} />}.
      </>
    </IssueActivityBlockComponent>
  );
};
