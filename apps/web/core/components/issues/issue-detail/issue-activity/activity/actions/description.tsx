/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { AlignLeft } from "lucide-react";
// hooks
import type { TIssueActivity } from "@plane/types";
// components
import { IssueActivityBlockComponent, IssueLink } from "./";

type TIssueDescriptionActivity = { activity: TIssueActivity; showIssue?: boolean; ends: "top" | "bottom" | undefined };

export const IssueDescriptionActivity = function IssueDescriptionActivity(props: TIssueDescriptionActivity) {
  const { activity, showIssue = true, ends } = props;
  // hooks

  if (!activity) return <></>;
  return (
    <IssueActivityBlockComponent
      icon={<AlignLeft size={14} className="text-secondary" aria-hidden="true" />}
      activity={activity}
      ends={ends}
    >
      <>
        updated the description
        {showIssue ? ` of ` : ``}
        {showIssue && <IssueLink activity={activity} />}.
      </>
    </IssueActivityBlockComponent>
  );
};
