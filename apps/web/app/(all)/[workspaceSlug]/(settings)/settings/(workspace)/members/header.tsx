/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { WORKSPACE_SETTINGS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
// components
import { SettingsBreadcrumbHeader } from "@/components/settings/page-header";
import { WORKSPACE_SETTINGS_ICONS } from "@/components/settings/workspace/sidebar/item-icon";

export const MembersWorkspaceSettingsHeader = function MembersWorkspaceSettingsHeader() {
  // plane hooks
  const { t } = useTranslation();
  // derived values
  const settingsDetails = WORKSPACE_SETTINGS.members;
  const Icon = WORKSPACE_SETTINGS_ICONS.members;

  return (
    <SettingsBreadcrumbHeader label={t(settingsDetails.i18n_label)} icon={<Icon className="size-4 text-tertiary" />} />
  );
};
