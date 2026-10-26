/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane types
import { WORKSPACE_SETTINGS } from "@plane/constants";
// components
import { useTranslation } from "@plane/i18n";
import type { TPowerKContext } from "@/components/power-k/core/types";
import { PowerKSettingsMenu } from "@/components/power-k/menus/settings";
import { WORKSPACE_SETTINGS_ICONS } from "@/components/settings/workspace/sidebar/item-icon";
// hooks
import { useWorkspacePermissions } from "@/hooks/use-permissions";

type Props = {
  context: TPowerKContext;
  handleSelect: (href: string) => void;
};

export function PowerKOpenWorkspaceSettingsMenu(props: Props) {
  const { context, handleSelect } = props;
  // plane hooks
  const { t } = useTranslation();
  // store hooks
  const workspaceSlug = context.params.workspaceSlug?.toString();
  const { can } = useWorkspacePermissions(workspaceSlug);
  // derived values
  const settingsList = Object.values(WORKSPACE_SETTINGS).filter(
    (setting) => context.params.workspaceSlug && can(setting.access)
  );
  const settingsListWithIcons = settingsList.map((setting) => ({
    ...setting,
    label: t(setting.i18n_label),
    icon: WORKSPACE_SETTINGS_ICONS[setting.key],
  }));

  return <PowerKSettingsMenu settings={settingsListWithIcons} onSelect={(setting) => handleSelect(setting.href)} />;
}
