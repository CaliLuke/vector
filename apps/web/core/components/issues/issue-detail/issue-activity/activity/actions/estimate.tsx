/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { EstimatePropertyIcon } from "@plane/propel/icons";
// hooks
import type { TIssueActivity } from "@plane/types";
// components
import { IssueActivityBlockComponent, IssueLink } from "./";

type TIssueEstimateActivity = { activity: TIssueActivity; showIssue?: boolean; ends: "top" | "bottom" | undefined };

export const IssueEstimateActivity = function IssueEstimateActivity(props: TIssueEstimateActivity) {
  const { activity, showIssue = true, ends } = props;
  // hooks

  if (!activity) return <></>;

  return (
    <IssueActivityBlockComponent
      icon={<EstimatePropertyIcon className="h-3.5 w-3.5 text-secondary" aria-hidden="true" />}
      activity={activity}
      ends={ends}
    >
      <>
        {activity.new_value ? `set the estimate point to ` : `removed the estimate point`}
        {!activity.new_value && activity.old_value && " "}
        {activity.new_value || activity.old_value}
        {showIssue && (activity.new_value ? ` to ` : ` from `)}
        {showIssue && <IssueLink activity={activity} />}.
      </>
    </IssueActivityBlockComponent>
  );
};
