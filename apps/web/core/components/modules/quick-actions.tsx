/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useLocation, useRouter } from "@tanstack/react-router";
import { useState } from "react";

import { MoreHorizontal } from "lucide-react";

// plane imports
import { EUserPermissions } from "@plane/constants";
import { IconButton } from "@plane/propel/icon-button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { TContextMenuItem } from "@plane/ui";
import { ContextMenu, CustomMenu } from "@plane/ui";
import { copyUrlToClipboard } from "@plane/utils/string";
import { cn } from "@plane/utils/common";
// components
import { useModuleMenuItems } from "@/components/common/quick-actions-helper";
import { ArchiveModuleModal, CreateUpdateModuleModal, DeleteModuleModal } from "@/components/modules";
// hooks
import { useProjectModules } from "@/hooks/use-modules";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import type { IModule } from "@plane/types";
import { useProjectPermissions } from "@/hooks/use-permissions";

type Props = {
  parentRef: React.RefObject<HTMLDivElement | null>;
  moduleId: string;
  module: IModule;
  projectId: string;
  workspaceSlug: string;
  customClassName?: string;
};

export const ModuleQuickActions = function ModuleQuickActions(props: Props) {
  const { parentRef, moduleId, projectId, workspaceSlug, customClassName } = props;
  // router
  const router = useRouter();
  const { moduleId: routeModuleId } = useParams({ strict: false });
  const peekModule = new URLSearchParams(useLocation({ select: (location) => location.searchStr })).get("peekModule");
  // states
  const [editModal, setEditModal] = useState(false);
  const [archiveModuleModal, setArchiveModuleModal] = useState(false);
  const [deleteModal, setDeleteModal] = useState(false);
  // store hooks
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);

  const { mutation, scope } = useProjectModules(workspaceSlug, projectId, Boolean(props.module.archived_at));
  const mounted = useCommittedScope(JSON.stringify([scope, moduleId]));

  // derived values
  const moduleDetails = props.module;
  // auth
  const isEditingAllowed = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  const moduleLink = `${workspaceSlug}/projects/${projectId}/modules/${moduleId}`;
  const handleCopyText = () =>
    copyUrlToClipboard(moduleLink).then(() => {
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Link Copied!",
        message: "Module link copied to clipboard.",
      });
      return undefined;
    });
  const handleOpenInNewTab = () => window.open(`/${moduleLink}`, "_blank");

  const handleRestoreModule = async () => {
    try {
      const result = await mutation.mutateAsync({ type: "restore", id: moduleId });
      if (!mounted() || !result.captured.isCurrent()) return;
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Restore success",
        message: "Your module can be found in project modules.",
      });
      void router.navigate({ href: `/${workspaceSlug}/projects/${projectId}/archives/modules` });
    } catch (_error) {
      if (!mounted()) return;
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Module could not be restored. Please try again.",
      });
    }
  };

  // Use unified menu hook from plane-web (resolves to CE or EE)
  const menuResult = useModuleMenuItems({
    moduleDetails: moduleDetails ?? undefined,
    workspaceSlug,
    projectId,
    moduleId,
    isEditingAllowed,
    handleEdit: () => setEditModal(true),
    handleArchive: () => setArchiveModuleModal(true),
    handleRestore: handleRestoreModule,
    handleDelete: () => setDeleteModal(true),
    handleCopyLink: handleCopyText,
    handleOpenInNewTab,
  });

  // Handle both CE (array) and EE (object) return types
  const MENU_ITEMS: TContextMenuItem[] = Array.isArray(menuResult) ? menuResult : menuResult.items;
  const additionalModals = Array.isArray(menuResult) ? null : menuResult.modals;

  const CONTEXT_MENU_ITEMS = MENU_ITEMS.map(function CONTEXT_MENU_ITEMS(item) {
    return {
      ...item,

      onClick: () => {
        item.action();
      },
    };
  });

  return (
    <>
      {moduleDetails && (
        <div className="fixed">
          <CreateUpdateModuleModal
            isOpen={editModal}
            onClose={() => setEditModal(false)}
            data={moduleDetails}
            projectId={projectId}
            workspaceSlug={workspaceSlug}
          />
          <ArchiveModuleModal
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            moduleId={moduleId}
            isOpen={archiveModuleModal}
            handleClose={() => setArchiveModuleModal(false)}
          />
          <DeleteModuleModal
            data={moduleDetails}
            isOpen={deleteModal}
            onClose={() => setDeleteModal(false)}
            onDeleteSuccess={(deletedModule) => {
              if (routeModuleId === deletedModule.id || peekModule === deletedModule.id) {
                void router.navigate({ href: `/${workspaceSlug}/projects/${deletedModule.project_id}/modules` });
              }
            }}
          />
          {additionalModals}
        </div>
      )}
      <ContextMenu parentRef={parentRef} items={CONTEXT_MENU_ITEMS} />
      <CustomMenu
        customButtonAsChild
        customButton={<IconButton variant="tertiary" size="lg" icon={MoreHorizontal} />}
        placement="bottom-end"
        closeOnSelect
        buttonClassName={customClassName}
      >
        {MENU_ITEMS.map((item) => {
          if (item.shouldRender === false) return null;
          return (
            <CustomMenu.MenuItem
              key={item.key}
              onClick={() => {
                item.action();
              }}
              className={cn(
                "flex items-center gap-2",
                {
                  "text-placeholder": item.disabled,
                },
                item.className
              )}
              disabled={item.disabled}
            >
              {item.icon && <item.icon className={cn("h-3 w-3 flex-shrink-0", item.iconClassName)} />}
              <div>
                <h5>{item.title}</h5>
                {item.description && (
                  <p
                    className={cn("whitespace-pre-line text-tertiary", {
                      "text-placeholder": item.disabled,
                    })}
                  >
                    {item.description}
                  </p>
                )}
              </div>
            </CustomMenu.MenuItem>
          );
        })}
      </CustomMenu>
    </>
  );
};
