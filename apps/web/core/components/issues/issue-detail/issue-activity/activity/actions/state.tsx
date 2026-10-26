/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// hooks
import { StatePropertyIcon } from "@plane/propel/icons";
import type { TIssueActivity } from "@plane/types";
// components
import { IssueActivityBlockComponent, IssueLink } from "./";
// icons

type TIssueStateActivity = { activity: TIssueActivity; showIssue?: boolean; ends: "top" | "bottom" | undefined };

export const IssueStateActivity = function IssueStateActivity(props: TIssueStateActivity) {
  const { activity, showIssue = true, ends } = props;
  // hooks

  if (!activity) return <></>;
  return (
    <IssueActivityBlockComponent
      icon={<StatePropertyIcon className="h-4 w-4 flex-shrink-0 text-secondary" />}
      activity={activity}
      ends={ends}
    >
      <>
        set the state to <span className="font-medium text-primary">{activity.new_value}</span>
        {showIssue ? ` for ` : ``}
        {showIssue && <IssueLink activity={activity} />}.
      </>
    </IssueActivityBlockComponent>
  );
};
