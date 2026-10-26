/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ParentPropertyIcon } from "@plane/propel/icons";
// hooks
import type { TIssueActivity } from "@plane/types";
// components
import { IssueActivityBlockComponent, IssueLink } from "./";

type TIssueParentActivity = { activity: TIssueActivity; showIssue?: boolean; ends: "top" | "bottom" | undefined };

export const IssueParentActivity = function IssueParentActivity(props: TIssueParentActivity) {
  const { activity, showIssue = true, ends } = props;
  // hooks

  if (!activity) return <></>;
  return (
    <IssueActivityBlockComponent
      icon={<ParentPropertyIcon className="h-3.5 w-3.5 text-secondary" aria-hidden="true" />}
      activity={activity}
      ends={ends}
    >
      <>
        {activity.new_value ? `set the parent to ` : `removed the parent `}
        {activity.new_value ? (
          <span className="font-medium text-primary">{activity.new_value}</span>
        ) : (
          <span className="font-medium text-primary">{activity.old_value}</span>
        )}
        {showIssue && (activity.new_value ? ` for ` : ` from `)}
        {showIssue && <IssueLink activity={activity} />}.
      </>
    </IssueActivityBlockComponent>
  );
};
