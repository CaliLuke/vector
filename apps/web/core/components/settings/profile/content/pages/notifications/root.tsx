/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery } from "@tanstack/react-query";
// plane imports
import { useTranslation } from "@plane/i18n";
// components
import { ProfileSettingsHeading } from "@/components/settings/profile/heading";
import { EmailSettingsLoader } from "@/components/ui/loader/settings/email";
import { useCurrentUser } from "@/hooks/use-current-user";
// services
import { emailNotificationSettingsOptions } from "@/queries/user-settings";
// local imports
import { NotificationsProfileSettingsForm } from "./email-notification-form";

export function NotificationsProfileSettings() {
  const { t } = useTranslation();
  const { data: user } = useCurrentUser();
  // fetching user email notification settings
  const { data, isLoading, isError, refetch } = useQuery(emailNotificationSettingsOptions(user?.id));
  if (isError)
    return (
      <div role="alert">
        Unable to load notification settings.{" "}
        <button
          type="button"
          onClick={() => {
            void refetch();
          }}
        >
          Retry
        </button>
      </div>
    );

  if (!data || isLoading) {
    return <EmailSettingsLoader />;
  }

  return (
    <div className="size-full">
      <ProfileSettingsHeading
        title={t("account_settings.notifications.heading")}
        description={t("account_settings.notifications.description")}
      />
      <div className="mt-7">
        <NotificationsProfileSettingsForm data={data} userId={user?.id} />
      </div>
    </div>
  );
}
