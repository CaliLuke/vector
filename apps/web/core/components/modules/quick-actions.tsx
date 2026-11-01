/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useLocation, useRouter } from "@tanstack/react-router";

// plane imports
import { EUserPermissions } from "@plane/constants";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { QuickActionsMenu } from "@/components/common/quick-actions-menu";
import { normalizeQuickActions, useQuickActionDialogs } from "@/components/common/quick-actions-helper";
import { copyUrlToClipboard } from "@plane/utils/string";
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
  const dialogs = useQuickActionDialogs();
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
    copyUrlToClipboard(moduleLink)
      .then(() => {
        if (!mounted()) return;
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: "Link Copied!",
          message: "Module link copied to clipboard.",
        });
        return;
      })
      .catch(() => {
        if (!mounted()) return;
        setToast({ type: TOAST_TYPE.ERROR, title: "Could not copy to clipboard", message: "Please try again." });
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
    ...dialogs.handlers,
    handleRestore: () => void handleRestoreModule(),
    handleCopyLink: () => void handleCopyText(),
    handleOpenInNewTab,
  });

  const { items: MENU_ITEMS, modals: additionalModals } = normalizeQuickActions(menuResult);

  return (
    <>
      {moduleDetails && (
        <div className="fixed">
          <CreateUpdateModuleModal
            isOpen={dialogs.edit.isOpen}
            onClose={dialogs.edit.close}
            data={moduleDetails}
            projectId={projectId}
            workspaceSlug={workspaceSlug}
          />
          <ArchiveModuleModal
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            moduleId={moduleId}
            isOpen={dialogs.archive.isOpen}
            handleClose={dialogs.archive.close}
          />
          <DeleteModuleModal
            data={moduleDetails}
            isOpen={dialogs.delete.isOpen}
            onClose={dialogs.delete.close}
            onDeleteSuccess={(deletedModule) => {
              if (routeModuleId === deletedModule.id || peekModule === deletedModule.id) {
                void router.navigate({ href: `/${workspaceSlug}/projects/${deletedModule.project_id}/modules` });
              }
            }}
          />
          {additionalModals}
        </div>
      )}
      <QuickActionsMenu parentRef={parentRef} items={MENU_ITEMS} className={customClassName} />
    </>
  );
};
