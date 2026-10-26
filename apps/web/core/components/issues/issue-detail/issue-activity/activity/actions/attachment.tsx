/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Paperclip } from "lucide-react";
// hooks
import type { TIssueActivity } from "@plane/types";
// components
import { IssueActivityBlockComponent, IssueLink } from "./";

type TIssueAttachmentActivity = { activity: TIssueActivity; showIssue?: boolean; ends: "top" | "bottom" | undefined };

export const IssueAttachmentActivity = function IssueAttachmentActivity(props: TIssueAttachmentActivity) {
  const { activity, showIssue = true, ends } = props;
  // hooks

  if (!activity) return <></>;
  return (
    <IssueActivityBlockComponent
      icon={<Paperclip size={14} className="text-secondary" aria-hidden="true" />}
      activity={activity}
      ends={ends}
    >
      <>
        {activity.verb === "created" ? `uploaded a new attachment` : `removed an attachment`}
        {showIssue && (activity.verb === "created" ? ` to ` : ` from `)}
        {showIssue && <IssueLink activity={activity} />}.
      </>
    </IssueActivityBlockComponent>
  );
};
