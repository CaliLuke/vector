/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Dispatch, SetStateAction } from "react";
import { Fragment } from "react";
import { Clock } from "lucide-react";
import { Popover, Transition } from "@headlessui/react";
// plane imports
import { NOTIFICATION_SNOOZE_OPTIONS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { Tooltip } from "@plane/propel/tooltip";
import { cn } from "@plane/utils/common";
// hooks
import { usePlatformOS } from "@/hooks/use-platform-os";
// store
import type { TNotification } from "@plane/types";
import { useUpdateNotification } from "@/queries/notifications";
// local imports
import { NotificationSnoozeModal } from "./modal";

type TNotificationItemSnoozeOption = {
  workspaceSlug: string;
  notification: TNotification;
  customSnoozeModal: boolean;
  setCustomSnoozeModal: Dispatch<SetStateAction<boolean>>;
};

export const NotificationItemSnoozeOption = function NotificationItemSnoozeOption(
  props: TNotificationItemSnoozeOption
) {
  const { workspaceSlug, notification, customSnoozeModal, setCustomSnoozeModal } = props;
  // hooks
  const { isMobile } = usePlatformOS();
  const { t } = useTranslation();
  const data = notification;
  const update = useUpdateNotification(workspaceSlug, notification.id);

  const handleNotificationSnoozeDate = async (snoozeTill: Date | undefined) => {
    await update.mutateAsync({ notification, field: "snoozed_till", value: snoozeTill?.toISOString() ?? null });
    setToast({
      title: `${t("common.success")}!`,
      message: t(snoozeTill ? "notification.toasts.snoozed" : "notification.toasts.unsnoozed"),
      type: TOAST_TYPE.SUCCESS,
    });
    setCustomSnoozeModal(false);
  };

  const handleDropdownSelect = async (snoozeDate: Date | "un-snooze" | undefined) => {
    if (update.isPending) return;
    if (snoozeDate === undefined) {
      setCustomSnoozeModal(true);
      return;
    }
    try {
      await handleNotificationSnoozeDate(snoozeDate === "un-snooze" ? undefined : snoozeDate);
    } catch {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("common.errors.default.title"),
        message: t("common.errors.default.message"),
      });
    }
  };

  return (
    <>
      <NotificationSnoozeModal
        isOpen={customSnoozeModal}
        onClose={() => setCustomSnoozeModal(false)}
        onSubmit={handleNotificationSnoozeDate}
      />
      <Popover className="relative">
        {({ open, close }) => (
          <>
            <Tooltip
              tooltipContent={
                data.snoozed_till ? t("notification.options.mark_unsnooze") : t("notification.options.mark_snooze")
              }
              isMobile={isMobile}
            >
              <Popover.Button
                disabled={update.isPending}
                className={cn(
                  "relative flex h-5 w-5 flex-shrink-0 items-center justify-center rounded-xs bg-layer-1 outline-none hover:bg-surface-2",
                  open ? "bg-layer-1" : ""
                )}
              >
                <Clock className="h-3 w-3 text-tertiary" />
              </Popover.Button>
            </Tooltip>

            <Transition
              as={Fragment}
              enter="transition ease-out duration-200"
              enterFrom="opacity-0 translate-y-1"
              enterTo="opacity-100 translate-y-0"
              leave="transition ease-in duration-150"
              leaveFrom="opacity-100 translate-y-0"
              leaveTo="opacity-0 translate-y-1"
            >
              <Popover.Panel className="absolute right-0 z-10 mt-2 min-w-44 select-none">
                <div className="space-y-1 rounded-md border border-subtle bg-surface-1 p-2">
                  {data.snoozed_till && (
                    <button
                      className="w-full cursor-pointer rounded-xs p-1 px-2 text-left text-body-xs-medium text-secondary transition-all hover:bg-layer-1"
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        close();
                        void handleDropdownSelect("un-snooze");
                      }}
                    >
                      <div>{t("notification.options.mark_unsnooze")}</div>
                    </button>
                  )}

                  {NOTIFICATION_SNOOZE_OPTIONS.map((option) => (
                    <button
                      key={option.key}
                      className="w-full cursor-pointer rounded-xs p-1 px-2 text-left text-body-xs-medium text-secondary transition-all hover:bg-layer-1"
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        close();
                        void handleDropdownSelect(option.value != undefined ? option.value() : option.value);
                      }}
                    >
                      <div>{t(option?.i18n_label)}</div>
                    </button>
                  ))}
                </div>
              </Popover.Panel>
            </Transition>
          </>
        )}
      </Popover>
    </>
  );
};
