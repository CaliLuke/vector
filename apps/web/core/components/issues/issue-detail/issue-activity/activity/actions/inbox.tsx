/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// hooks
import { IntakeIcon } from "@plane/propel/icons";
import type { TIssueActivity } from "@plane/types";
// components
import { IssueActivityBlockComponent } from "./";
// icons

type TIssueInboxActivity = { activity: TIssueActivity; ends: "top" | "bottom" | undefined };

export const IssueInboxActivity = function IssueInboxActivity(props: TIssueInboxActivity) {
  const { activity, ends } = props;
  // hooks

  const getInboxActivityMessage = () => {
    switch (activity?.verb) {
      case "-1":
        return "declined this work item from intake.";
      case "0":
        return "snoozed this work item.";
      case "1":
        return "accepted this work item from intake.";
      case "2":
        return "declined this work item from intake by marking a duplicate work item.";
      default:
        return "updated intake work item status.";
    }
  };

  if (!activity) return <></>;
  return (
    <IssueActivityBlockComponent
      icon={<IntakeIcon className="h-4 w-4 flex-shrink-0 text-secondary" />}
      activity={activity}
      ends={ends}
    >
      <>{getInboxActivityMessage()}</>
    </IssueActivityBlockComponent>
  );
};
