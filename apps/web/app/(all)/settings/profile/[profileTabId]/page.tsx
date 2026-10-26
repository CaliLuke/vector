/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute, useRouter } from "@tanstack/react-router";

// plane imports
import { PROFILE_SETTINGS_TABS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import type { TProfileSettingsTabs } from "@plane/types";
// components
import { LogoSpinner } from "@/components/common/logo-spinner";
import { PageHead } from "@/components/core/page-title";
import { ProfileSettingsContent } from "@/components/settings/profile/content";
import { ProfileSettingsSidebarRoot } from "@/components/settings/profile/sidebar";
// hooks
import { useCurrentUser } from "@/hooks/use-current-user";

// local imports

function ProfileSettingsPage(): React.JSX.Element | null {
  const { profileTabId } = Route.useParams();
  // router
  const router = useRouter();
  // store hooks
  const { data: currentUser } = useCurrentUser();
  // translation
  const { t } = useTranslation();
  // derived values
  const isAValidTab = PROFILE_SETTINGS_TABS.includes(profileTabId as TProfileSettingsTabs);

  if (!currentUser || !isAValidTab)
    return (
      <div className="grid size-full place-items-center px-4">
        <LogoSpinner />
      </div>
    );

  return (
    <>
      <PageHead title={`${t("profile.label")} - ${t("general_settings")}`} />
      <div className="relative size-full">
        <div className="flex size-full">
          <ProfileSettingsSidebarRoot
            activeTab={profileTabId as TProfileSettingsTabs}
            className="w-[250px]"
            updateActiveTab={(tab) => void router.navigate({ href: `/settings/profile/${tab}` })}
          />
          <ProfileSettingsContent
            activeTab={profileTabId as TProfileSettingsTabs}
            className="mx-auto w-fit max-w-225 grow px-page-x py-20"
          />
        </div>
      </div>
    </>
  );
}

export const Route = createFileRoute("/_authenticated/settings/profile/$profileTabId/")({
  component: ProfileSettingsPage,
});
