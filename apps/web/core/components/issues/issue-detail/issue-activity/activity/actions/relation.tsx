/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// types
import type { TIssueRelationTypes } from "@plane/types";
// hooks
import type { TIssueActivity } from "@plane/types";
// components
import { useTimeLineRelationOptions } from "@/components/relations";
// local helpers
import { IssueActivityBlockComponent } from "./";
import { getRelationActivityContent } from "./helpers/activity";

type TIssueRelationActivity = { activity: TIssueActivity; ends: "top" | "bottom" | undefined };

export const IssueRelationActivity = function IssueRelationActivity(props: TIssueRelationActivity) {
  const { activity, ends } = props;
  // hooks

  const ISSUE_RELATION_OPTIONS = useTimeLineRelationOptions();
  const activityContent = getRelationActivityContent(activity);

  if (!activity) return <></>;
  return (
    <IssueActivityBlockComponent
      icon={activity.field ? ISSUE_RELATION_OPTIONS[activity.field as TIssueRelationTypes]?.icon(14) : <></>}
      activity={activity}
      ends={ends}
    >
      {activityContent}
      {activity.old_value === "" ? (
        <span className="font-medium text-primary">{activity.new_value}.</span>
      ) : (
        <span className="font-medium text-primary">{activity.old_value}.</span>
      )}
    </IssueActivityBlockComponent>
  );
};
