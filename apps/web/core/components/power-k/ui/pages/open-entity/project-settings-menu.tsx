/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { PROJECT_SETTINGS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
// components
import type { TPowerKContext } from "@/components/power-k/core/types";
import { PowerKSettingsMenu } from "@/components/power-k/menus/settings";
import { PROJECT_SETTINGS_ICONS } from "@/components/settings/project/sidebar/item-icon";
// hooks
import { useProjectPermissions } from "@/hooks/use-permissions";

type Props = {
  context: TPowerKContext;
  handleSelect: (href: string) => void;
};

export function PowerKOpenProjectSettingsMenu(props: Props) {
  const { context, handleSelect } = props;
  // plane hooks
  const { t } = useTranslation();
  // store hooks
  const workspaceSlug = context.params.workspaceSlug?.toString();
  const projectId = context.params.projectId?.toString();
  const { can } = useProjectPermissions(workspaceSlug, projectId);
  // derived values
  const settingsList = Object.values(PROJECT_SETTINGS).filter(
    (setting) => context.params.workspaceSlug && context.params.projectId && can(setting.access)
  );
  const settingsListWithIcons = settingsList.map((setting) => ({
    ...setting,
    label: t(setting.i18n_label),
    icon: PROJECT_SETTINGS_ICONS[setting.key],
  }));

  return <PowerKSettingsMenu settings={settingsListWithIcons} onSelect={(setting) => handleSelect(setting.href)} />;
}
