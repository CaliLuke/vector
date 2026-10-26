/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TProjectActivity } from "@plane/types";
import { useWorkspaceById } from "@/hooks/use-workspaces";
import { ActivityBlockComponent } from "./activity-block";
import { iconsMap, messages } from "./helper";

type TActivityItem = {
  activity: TProjectActivity;
  showProject?: boolean;
  ends?: "top" | "bottom" | undefined;
};

export function ActivityItem(props: TActivityItem) {
  const { activity, ends } = props;
  const { data: workspaceDetail } = useWorkspaceById(activity.workspace);

  if (!activity) return null;

  const activityType = activity.field;
  if (!activityType) return null;

  const { message, customUserName } = messages(activity, workspaceDetail);
  const icon = iconsMap[activityType] || iconsMap.default;

  return (
    <ActivityBlockComponent icon={icon} activity={activity} ends={ends} customUserName={customUserName}>
      <>{message}</>
    </ActivityBlockComponent>
  );
}
