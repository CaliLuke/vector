/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback } from "react";

import { Star, StarOff } from "lucide-react";
// plane imports
import { useTranslation } from "@plane/i18n";
import { LinkIcon } from "@plane/propel/icons";
import { setToast, TOAST_TYPE } from "@plane/propel/toast";
import { copyTextToClipboard } from "@plane/utils/string";
// components
import type { TPowerKCommandConfig } from "@/components/power-k/core/types";
// hooks
import { EUserProjectRoles } from "@plane/types";
import { useCycleDetails } from "@/hooks/use-cycles";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useFavorites } from "@/hooks/use-favorites";
import { useParams } from "@tanstack/react-router";

export const usePowerKCycleContextBasedActions = (): TPowerKCommandConfig[] => {
  // navigation
  const { workspaceSlug, projectId, cycleId } = useParams({ strict: false });
  // store
  const { cycle: cycleDetails } = useCycleDetails(workspaceSlug, projectId, cycleId);
  // derived values
  const { favoritesByEntity, toggleFavorite } = useFavorites(workspaceSlug?.toString());
  const isFavorite = favoritesByEntity?.has(`cycle:${cycleDetails?.id}`) ?? false;
  // permission
  const { can } = useProjectPermissions(workspaceSlug?.toString(), cycleDetails?.project_id);
  const isEditingAllowed = can([EUserProjectRoles.ADMIN, EUserProjectRoles.MEMBER]) && !cycleDetails?.archived_at;
  // translation
  const { t } = useTranslation();

  const toggleCycleFavorite = useCallback(() => {
    if (!workspaceSlug || !cycleDetails || !cycleDetails.project_id) return;
    void toggleFavorite("cycle", cycleDetails.id, cycleDetails.project_id).catch(() => {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Some error occurred",
      });
    });
  }, [toggleFavorite, workspaceSlug, cycleDetails]);

  const copyCycleUrlToClipboard = useCallback(() => {
    const url = new URL(window.location.href);
    copyTextToClipboard(url.href)
      .then(() => {
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("power_k.contextual_actions.cycle.copy_url_toast_success"),
        });
      })
      .catch(() => {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("power_k.contextual_actions.cycle.copy_url_toast_error"),
        });
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return [
    {
      id: "toggle_cycle_favorite",
      i18n_title: isFavorite
        ? "power_k.contextual_actions.cycle.remove_from_favorites"
        : "power_k.contextual_actions.cycle.add_to_favorites",
      icon: isFavorite ? StarOff : Star,
      group: "contextual",
      contextType: "cycle",
      type: "action",
      action: toggleCycleFavorite,
      modifierShortcut: "shift+f",
      isEnabled: () => isEditingAllowed,
      isVisible: () => isEditingAllowed,
      closeOnSelect: true,
    },
    {
      id: "copy_cycle_url",
      i18n_title: "power_k.contextual_actions.cycle.copy_url",
      icon: LinkIcon,
      group: "contextual",
      contextType: "cycle",
      type: "action",
      action: copyCycleUrlToClipboard,
      modifierShortcut: "cmd+shift+,",
      isEnabled: () => true,
      isVisible: () => true,
      closeOnSelect: true,
    },
  ];
};
