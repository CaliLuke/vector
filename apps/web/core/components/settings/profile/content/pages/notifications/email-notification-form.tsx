/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IUserEmailNotificationSettings } from "@plane/types";
import { ToggleSwitch } from "@plane/ui";
import { SettingsControlItem } from "@/components/settings/control-item";
import { emailNotificationSettingsMutationOptions } from "@/queries/user-settings";

const settings: { key: keyof IUserEmailNotificationSettings; title: string; description: string }[] = [
  { key: "property_change", title: "property_changes", description: "property_changes_description" },
  { key: "state_change", title: "state_change", description: "state_change_description" },
  { key: "issue_completed", title: "issue_completed", description: "issue_completed_description" },
  { key: "comment", title: "comments", description: "comments_description" },
  { key: "mention", title: "mentions", description: "mentions_description" },
];

export function NotificationsProfileSettingsForm({
  data,
  userId,
}: {
  data: IUserEmailNotificationSettings;
  userId: string | undefined;
}) {
  const { t } = useTranslation();
  const mutation = useMutation(emailNotificationSettingsMutationOptions(useQueryClient(), userId));
  const update = (key: keyof IUserEmailNotificationSettings, value: boolean) =>
    mutation.mutate(
      { [key]: value },
      {
        onSuccess: () =>
          setToast({
            title: t("success"),
            type: TOAST_TYPE.SUCCESS,
            message: t("email_notification_setting_updated_successfully"),
          }),
        onError: () =>
          setToast({
            title: t("error"),
            type: TOAST_TYPE.ERROR,
            message: t("failed_to_update_email_notification_setting"),
          }),
      }
    );
  return (
    <div className="flex flex-col gap-y-1">
      {settings.map(({ key, title, description }) => (
        <div key={key} className={key === "issue_completed" ? "border-l-3 border-subtle-1 pl-3" : undefined}>
          <SettingsControlItem
            title={t(title)}
            description={t(description)}
            control={
              <ToggleSwitch
                value={data[key]}
                onChange={(value) => update(key, value)}
                disabled={mutation.isPending}
                size="sm"
              />
            }
          />
        </div>
      ))}
    </div>
  );
}
