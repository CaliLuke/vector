/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { PROJECT_SETTINGS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
// components
import { SettingsBreadcrumbHeader } from "@/components/settings/page-header";
import { PROJECT_SETTINGS_ICONS } from "@/components/settings/project/sidebar/item-icon";

export const FeaturesCyclesProjectSettingsHeader = function FeaturesCyclesProjectSettingsHeader() {
  // translation
  const { t } = useTranslation();
  // derived values
  const settingsDetails = PROJECT_SETTINGS.features_cycles;
  const Icon = PROJECT_SETTINGS_ICONS.features_cycles;

  return (
    <SettingsBreadcrumbHeader label={t(settingsDetails.i18n_label)} icon={<Icon className="size-4 text-tertiary" />} />
  );
};
