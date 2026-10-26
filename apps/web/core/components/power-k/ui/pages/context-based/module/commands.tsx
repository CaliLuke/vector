/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useCallback } from "react";

import { Star, StarOff, Users } from "lucide-react";
// plane imports
import { useTranslation } from "@plane/i18n";
import { LinkIcon, ModuleStatusIcon } from "@plane/propel/icons";
import { setToast, TOAST_TYPE } from "@plane/propel/toast";
import type { IModule } from "@plane/types";
import { EUserProjectRoles } from "@plane/types";
import { copyTextToClipboard } from "@plane/utils/string";
// components
import type { TPowerKCommandConfig } from "@/components/power-k/core/types";
// hooks
import { useModuleDetails, useProjectModules } from "@/hooks/use-modules";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { MODULE_STATUS } from "@plane/constants";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useFavorites } from "@/hooks/use-favorites";

export const usePowerKModuleContextBasedActions = (): TPowerKCommandConfig[] => {
  // navigation
  const { workspaceSlug, projectId, moduleId } = useParams({ strict: false });
  // store
  const { module: moduleDetails } = useModuleDetails(workspaceSlug, projectId, moduleId);
  const catalogue = useProjectModules(workspaceSlug, projectId);
  const current = useCommittedScope(JSON.stringify([catalogue.scope, moduleId]));
  // derived values
  const { favoritesByEntity, toggleFavorite } = useFavorites(workspaceSlug?.toString());
  const isFavorite = favoritesByEntity?.has(`module:${moduleDetails?.id}`) ?? false;
  // permission
  const { can } = useProjectPermissions(workspaceSlug?.toString(), moduleDetails?.project_id);
  const isEditingAllowed = can([EUserProjectRoles.ADMIN, EUserProjectRoles.MEMBER]) && !moduleDetails?.archived_at;
  // translation
  const { t } = useTranslation();

  const handleUpdateModule = useCallback(
    async (formData: Partial<IModule>) => {
      if (!workspaceSlug || !projectId || !moduleDetails) return;
      await catalogue.mutation.mutateAsync({ type: "patch", id: moduleDetails.id, data: formData }).catch(() => {
        if (current())
          setToast({
            type: TOAST_TYPE.ERROR,
            title: "Error!",
            message: "Module could not be updated. Please try again.",
          });
      });
    },
    [moduleDetails, projectId, catalogue.mutation, workspaceSlug, current]
  );

  const handleUpdateMember = useCallback(
    (memberId: string) => {
      if (!moduleDetails) return;

      const updatedMembers = [...(moduleDetails.member_ids ?? [])];
      if (updatedMembers.includes(memberId)) updatedMembers.splice(updatedMembers.indexOf(memberId), 1);
      else updatedMembers.push(memberId);

      handleUpdateModule({ member_ids: updatedMembers });
    },
    [handleUpdateModule, moduleDetails]
  );

  const toggleModuleFavorite = useCallback(() => {
    if (!workspaceSlug || !moduleDetails || !moduleDetails.project_id) return;
    void toggleFavorite("module", moduleDetails.id, moduleDetails.project_id).catch(() => {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Some error occurred",
      });
    });
  }, [toggleFavorite, workspaceSlug, moduleDetails]);

  const copyModuleUrlToClipboard = useCallback(() => {
    const url = new URL(window.location.href);
    copyTextToClipboard(url.href)
      .then(() => {
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("power_k.contextual_actions.module.copy_url_toast_success"),
        });
      })
      .catch(() => {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("power_k.contextual_actions.module.copy_url_toast_error"),
        });
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return [
    {
      id: "add_remove_module_members",
      i18n_title: "power_k.contextual_actions.module.add_remove_members",
      icon: Users,
      group: "contextual",
      contextType: "module",
      type: "change-page",
      page: "update-module-member",
      onSelect: (data) => {
        if (typeof data === "string") handleUpdateMember(data);
      },
      shortcut: "m",
      isEnabled: () => isEditingAllowed,
      isVisible: () => isEditingAllowed,
      closeOnSelect: false,
    },
    {
      id: "change_module_status",
      i18n_title: "power_k.contextual_actions.module.change_status",
      iconNode: <ModuleStatusIcon status="backlog" className="size-3.5 shrink-0" />,
      group: "contextual",
      contextType: "module",
      type: "change-page",
      page: "update-module-status",
      onSelect: (data) => {
        const status = MODULE_STATUS.find((row) => row.value === data)?.value;
        if (status) void handleUpdateModule({ status });
      },
      shortcut: "s",
      isEnabled: () => isEditingAllowed,
      isVisible: () => isEditingAllowed,
      closeOnSelect: true,
    },
    {
      id: "toggle_module_favorite",
      i18n_title: isFavorite
        ? "power_k.contextual_actions.module.remove_from_favorites"
        : "power_k.contextual_actions.module.add_to_favorites",
      icon: isFavorite ? StarOff : Star,
      group: "contextual",
      contextType: "module",
      type: "action",
      action: toggleModuleFavorite,
      modifierShortcut: "shift+f",
      isEnabled: () => isEditingAllowed,
      isVisible: () => isEditingAllowed,
      closeOnSelect: true,
    },
    {
      id: "copy_module_url",
      i18n_title: "power_k.contextual_actions.module.copy_url",
      icon: LinkIcon,
      group: "contextual",
      contextType: "module",
      type: "action",
      action: copyModuleUrlToClipboard,
      modifierShortcut: "cmd+shift+,",
      isEnabled: () => true,
      isVisible: () => true,
      closeOnSelect: true,
    },
  ];
};
