/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Dispatch, SetStateAction } from "react";
// plane imports
import { cn } from "@plane/utils/common";
// hooks
import { useNotifications } from "@/components/workspace-notifications/provider";
// local imports
import { NotificationItemToggle } from "./toggle";
import { NotificationItemSnoozeOption } from "./snooze";

type TNotificationOption = {
  workspaceSlug: string;
  notificationId: string;
  customSnoozeModal: boolean;
  setCustomSnoozeModal: Dispatch<SetStateAction<boolean>>;
};

export const NotificationOption = function NotificationOption(props: TNotificationOption) {
  const { workspaceSlug, notificationId, customSnoozeModal, setCustomSnoozeModal } = props;
  // hooks
  const { notifications } = useNotifications();
  const notification = notifications.get(notificationId);
  if (!notification) return null;

  return (
    <div
      className={cn(
        "hidden flex-shrink-0 text-body-xs-medium group-hover:block focus-within:block has-[[data-open]]:block",
        {
          block: customSnoozeModal,
        }
      )}
    >
      <div className="relative flex items-center justify-center gap-2">
        {/* read */}
        <NotificationItemToggle field="read_at" workspaceSlug={workspaceSlug} notification={notification} />

        {/* archive */}
        <NotificationItemToggle field="archived_at" workspaceSlug={workspaceSlug} notification={notification} />

        {/* snooze notification */}
        <NotificationItemSnoozeOption
          workspaceSlug={workspaceSlug}
          notification={notification}
          customSnoozeModal={customSnoozeModal}
          setCustomSnoozeModal={setCustomSnoozeModal}
        />
      </div>
    </div>
  );
};
