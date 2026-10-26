/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ArchiveRestore, MessageSquare } from "lucide-react";
import { ArchiveIcon } from "@plane/propel/icons";
// plane imports
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
// store
import type { TNotification } from "@plane/types";
import { useUpdateNotification } from "@/queries/notifications";
// local imports
import { NotificationItemOptionButton } from "./button";

type NotificationToggleProps = {
  field: "read_at" | "archived_at";
  workspaceSlug: string;
  notification: TNotification;
};

const toggleMessages = {
  read_at: {
    on: "notification.toasts.read",
    off: "notification.toasts.unread",
    enable: "notification.options.mark_read",
    disable: "notification.options.mark_unread",
  },
  archived_at: {
    on: "notification.toasts.archived",
    off: "notification.toasts.unarchived",
    enable: "notification.options.mark_archive",
    disable: "notification.options.mark_unarchive",
  },
} as const;

export function NotificationItemToggle({ workspaceSlug, notification, field }: NotificationToggleProps) {
  // hooks
  const active = Boolean(notification[field]);
  const messages = toggleMessages[field];
  const Icon = field === "read_at" ? MessageSquare : active ? ArchiveRestore : ArchiveIcon;
  const update = useUpdateNotification(workspaceSlug, notification.id);
  const { t } = useTranslation();

  const handleNotificationUpdate = async () => {
    if (update.isPending) return;
    try {
      await update.mutateAsync({
        notification,
        field,
        value: active ? null : new Date().toISOString(),
      });
      setToast({
        title: t(active ? messages.off : messages.on),
        type: TOAST_TYPE.SUCCESS,
      });
    } catch (e) {
      console.error(e);
    }
  };

  return (
    <NotificationItemOptionButton
      tooltipContent={t(active ? messages.disable : messages.enable)}
      disabled={update.isPending}
      callBack={() => {
        void handleNotificationUpdate();
      }}
    >
      <Icon className="h-3 w-3 text-tertiary" />
    </NotificationItemOptionButton>
  );
}
