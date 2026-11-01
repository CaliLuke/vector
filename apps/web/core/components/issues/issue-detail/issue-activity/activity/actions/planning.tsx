/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import type { TIssueActivity } from "@plane/types";
import { IssueActivityBlockComponent } from "./helpers/activity-block";

export type PlanningActivityProps = {
  activity: TIssueActivity;
  ends: "top" | "bottom" | undefined;
};

export function PlanningActivity({
  activity,
  ends,
  kind,
  icon,
}: PlanningActivityProps & { kind: "cycle" | "module"; icon: ReactNode }) {
  if (!activity) return null;
  const created = activity.verb === "created";
  const removed = !created && activity.verb !== "updated";
  const identifier = removed ? activity.old_identifier : activity.new_identifier;
  const value = removed ? activity.old_value : activity.new_value;
  const message = created
    ? `added this work item to the ${kind} `
    : removed
      ? `removed the work item from the ${kind} `
      : `set the ${kind} to `;
  return (
    <IssueActivityBlockComponent icon={icon} activity={activity} ends={ends}>
      <span>{message}</span>
      <a
        href={`/${activity.workspace_detail?.slug}/projects/${activity.project}/${kind}s/${identifier}`}
        target="_blank"
        rel="noopener noreferrer"
        className="inline-flex items-center gap-1 truncate font-medium text-primary hover:underline"
      >
        <span className="truncate">
          {!created && " "}
          {value}
        </span>
      </a>
    </IssueActivityBlockComponent>
  );
}
