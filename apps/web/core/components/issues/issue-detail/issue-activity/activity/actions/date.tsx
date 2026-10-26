/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { CalendarDays } from "lucide-react";
// hooks
import { renderFormattedDate } from "@plane/utils/datetime";
import type { TIssueActivity } from "@plane/types";
// components
import { IssueActivityBlockComponent, IssueLink } from "./";
// helpers

type IssueDateActivityProps = { activity: TIssueActivity; showIssue?: boolean; ends: "top" | "bottom" | undefined };

export const IssueDateActivity = function IssueDateActivity(props: IssueDateActivityProps) {
  const { activity, showIssue = true, ends } = props;
  const label = activity.field === "start_date" ? "start date" : "due date";
  if (!activity) return <></>;
  return (
    <IssueActivityBlockComponent
      icon={<CalendarDays size={14} className="text-secondary" aria-hidden="true" />}
      activity={activity}
      ends={ends}
    >
      <>
        {activity.new_value ? `set the ${label} to ` : `removed the ${label} `}
        {activity.new_value && (
          <>
            <span className="font-medium text-primary">{renderFormattedDate(activity.new_value)}</span>
          </>
        )}
        {showIssue && (activity.new_value ? ` for ` : ` from `)}
        {showIssue && <IssueLink activity={activity} />}.
      </>
    </IssueActivityBlockComponent>
  );
};
