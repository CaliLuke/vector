/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { RotateCcw } from "lucide-react";
// hooks
import { ArchiveIcon } from "@plane/propel/icons";
import type { TIssueActivity } from "@plane/types";
// components
import { IssueActivityBlockComponent } from "./";
// ui

type TIssueArchivedAtActivity = { activity: TIssueActivity; ends: "top" | "bottom" | undefined };

export const IssueArchivedAtActivity = function IssueArchivedAtActivity(props: TIssueArchivedAtActivity) {
  const { activity, ends } = props;
  // hooks

  if (!activity) return <></>;

  return (
    <IssueActivityBlockComponent
      icon={
        activity.new_value === "restore" ? (
          <RotateCcw className="h-3.5 w-3.5 text-secondary" aria-hidden="true" />
        ) : (
          <ArchiveIcon className="h-3.5 w-3.5 text-secondary" aria-hidden="true" />
        )
      }
      activity={activity}
      ends={ends}
      customUserName={activity.new_value === "archive" ? "Vector" : undefined}
    >
      {activity.new_value === "restore" ? "restored the work item" : "archived the work item"}.
    </IssueActivityBlockComponent>
  );
};
