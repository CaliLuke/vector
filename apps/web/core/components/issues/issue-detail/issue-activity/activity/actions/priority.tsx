/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { PriorityPropertyIcon } from "@plane/propel/icons";
// hooks
import type { TIssueActivity } from "@plane/types";
// components
import { IssueActivityBlockComponent, IssueLink } from "./";

type TIssuePriorityActivity = { activity: TIssueActivity; showIssue?: boolean; ends: "top" | "bottom" | undefined };

export const IssuePriorityActivity = function IssuePriorityActivity(props: TIssuePriorityActivity) {
  const { activity, showIssue = true, ends } = props;
  // hooks

  if (!activity) return <></>;
  return (
    <IssueActivityBlockComponent
      icon={<PriorityPropertyIcon className="h-3.5 w-3.5 text-secondary" aria-hidden="true" />}
      activity={activity}
      ends={ends}
    >
      <>
        set the priority to <span className="font-medium text-primary">{activity.new_value}</span>
        {showIssue ? ` for ` : ``}
        {showIssue && <IssueLink activity={activity} />}.
      </>
    </IssueActivityBlockComponent>
  );
};
