/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// hooks
import { CycleIcon } from "@plane/propel/icons";
import type { TIssueActivity } from "@plane/types";
// components
import { IssueActivityBlockComponent } from "./";
// icons

type TIssueCycleActivity = { activity: TIssueActivity; ends: "top" | "bottom" | undefined };

export const IssueCycleActivity = function IssueCycleActivity(props: TIssueCycleActivity) {
  const { activity, ends } = props;
  // hooks

  if (!activity) return <></>;
  return (
    <IssueActivityBlockComponent
      icon={<CycleIcon className="h-4 w-4 flex-shrink-0 text-secondary" />}
      activity={activity}
      ends={ends}
    >
      <>
        {activity.verb === "created" ? (
          <>
            <span>added this work item to the cycle </span>
            <a
              href={`/${activity.workspace_detail?.slug}/projects/${activity.project}/cycles/${activity.new_identifier}`}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1 truncate font-medium text-primary hover:underline"
            >
              <span className="truncate">{activity.new_value}</span>
            </a>
          </>
        ) : activity.verb === "updated" ? (
          <>
            <span>set the cycle to </span>
            <a
              href={`/${activity.workspace_detail?.slug}/projects/${activity.project}/cycles/${activity.new_identifier}`}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1 truncate font-medium text-primary hover:underline"
            >
              <span className="truncate"> {activity.new_value}</span>
            </a>
          </>
        ) : (
          <>
            <span>removed the work item from the cycle </span>
            <a
              href={`/${activity.workspace_detail?.slug}/projects/${activity.project}/cycles/${activity.old_identifier}`}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1 truncate font-medium text-primary hover:underline"
            >
              <span className="truncate"> {activity.new_value}</span>
            </a>
          </>
        )}
      </>
    </IssueActivityBlockComponent>
  );
};
