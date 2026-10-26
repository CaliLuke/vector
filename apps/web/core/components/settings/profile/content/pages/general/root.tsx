/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTranslation } from "@plane/i18n";
// components
import { PageHead } from "@/components/core/page-title";
// hooks
import { useCurrentUser } from "@/hooks/use-current-user";
// local imports
import { GeneralProfileSettingsForm } from "./form";

export function GeneralProfileSettings() {
  const { t } = useTranslation();
  // store hooks
  const { data: currentUser } = useCurrentUser();

  if (!currentUser) return null;

  return (
    <>
      <PageHead title={`${t("profile.label")} - ${t("general_settings")}`} />
      <GeneralProfileSettingsForm key={currentUser.id} user={currentUser} />
    </>
  );
}
