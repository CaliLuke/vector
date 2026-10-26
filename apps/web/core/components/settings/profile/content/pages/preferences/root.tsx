/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { useTranslation } from "@plane/i18n";
// components
import { ProfileSettingsHeading } from "@/components/settings/profile/heading";
// hooks
import { useCurrentUserProfile } from "@/hooks/use-current-user-profile";
// local imports
import { ProfileSettingsDefaultPreferencesList } from "./default-list";
import { ProfileSettingsLanguageAndTimezonePreferencesList } from "./language-and-timezone-list";

export const PreferencesProfileSettings = function PreferencesProfileSettings() {
  const { t } = useTranslation();
  // hooks
  const { data: userProfile } = useCurrentUserProfile();

  if (!userProfile) return null;

  return (
    <div className="size-full">
      <ProfileSettingsHeading
        title={t("account_settings.preferences.heading")}
        description={t("account_settings.preferences.description")}
      />
      <div className="mt-7 flex w-full flex-col gap-6">
        <section>
          <ProfileSettingsDefaultPreferencesList />
        </section>
        <section className="flex flex-col gap-y-3">
          <div className="text-h6-medium text-primary">{t("language_and_time")}</div>
          <ProfileSettingsLanguageAndTimezonePreferencesList />
        </section>
      </div>
    </div>
  );
};
