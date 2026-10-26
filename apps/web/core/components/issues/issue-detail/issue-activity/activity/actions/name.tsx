/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Type } from "lucide-react";
// hooks
import type { TIssueActivity } from "@plane/types";
// components
import { IssueActivityBlockComponent } from "./";

type TIssueNameActivity = { activity: TIssueActivity; ends: "top" | "bottom" | undefined };

export const IssueNameActivity = function IssueNameActivity(props: TIssueNameActivity) {
  const { activity, ends } = props;
  // hooks

  if (!activity) return <></>;
  return (
    <IssueActivityBlockComponent
      icon={<Type size={14} className="text-secondary" aria-hidden="true" />}
      activity={activity}
      ends={ends}
    >
      <>set the name to {activity.new_value}.</>
    </IssueActivityBlockComponent>
  );
};
