/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRouter } from "@tanstack/react-router";
import { useState } from "react";

import { MoreHorizontal } from "lucide-react";
// ui
import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { IconButton } from "@plane/propel/icon-button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { TContextMenuItem } from "@plane/ui";
import { ContextMenu, CustomMenu } from "@plane/ui";
import { copyUrlToClipboard } from "@plane/utils/string";
import { cn } from "@plane/utils/common";
// hooks
import { useCycleMenuItems } from "@/components/common/quick-actions-helper";
import { useProjectCycles } from "@/hooks/use-cycles";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import type { ICycle } from "@plane/types";
import { useProjectPermissions } from "@/hooks/use-permissions";

// local imports
import { ArchiveCycleModal } from "./archived-cycles/modal";
import { CycleDeleteModal } from "./delete-modal";
import { CycleCreateUpdateModal } from "./modal";

type Props = {
  parentRef: React.RefObject<HTMLElement | null>;
  cycleId: string;
  cycle: ICycle;
  projectId: string;
  workspaceSlug: string;
  customClassName?: string;
};

export const CycleQuickActions = function CycleQuickActions(props: Props) {
  const { parentRef, cycleId, projectId, workspaceSlug, customClassName } = props;
  // router
  const router = useRouter();
  // states
  const [updateModal, setUpdateModal] = useState(false);
  const [archiveCycleModal, setArchiveCycleModal] = useState(false);
  const [deleteModal, setDeleteModal] = useState(false);
  // store hooks
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  const { mutation, scope } = useProjectCycles(workspaceSlug, projectId, Boolean(props.cycle.archived_at));
  const mounted = useCommittedScope(JSON.stringify([scope, cycleId]));
  const { t } = useTranslation();
  // derived values
  const cycleDetails = props.cycle;
  // auth
  const isEditingAllowed = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  const cycleLink = `${workspaceSlug}/projects/${projectId}/cycles/${cycleId}`;
  const handleCopyText = () =>
    copyUrlToClipboard(cycleLink).then(() => {
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("common.link_copied"),
        message: t("common.link_copied_to_clipboard"),
      });
      return undefined;
    });
  const handleOpenInNewTab = () => window.open(`/${cycleLink}`, "_blank");

  const handleRestoreCycle = async () => {
    try {
      const result = await mutation.mutateAsync({ type: "restore", id: cycleId });
      if (!mounted() || !result.captured.isCurrent()) return;
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("project_cycles.action.restore.success.title"),
        message: t("project_cycles.action.restore.success.description"),
      });
      void router.navigate({ href: `/${workspaceSlug}/projects/${projectId}/archives/cycles` });
    } catch {
      if (!mounted()) return;
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("project_cycles.action.restore.failed.title"),
        message: t("project_cycles.action.restore.failed.description"),
      });
    }
  };

  const menuResult = useCycleMenuItems({
    cycleDetails: cycleDetails ?? undefined,
    workspaceSlug,
    projectId,
    cycleId,
    isEditingAllowed,
    handleEdit: () => setUpdateModal(true),
    handleArchive: () => setArchiveCycleModal(true),
    handleRestore: handleRestoreCycle,
    handleDelete: () => setDeleteModal(true),
    handleCopyLink: handleCopyText,
    handleOpenInNewTab,
  });

  const MENU_ITEMS: TContextMenuItem[] = Array.isArray(menuResult) ? menuResult : menuResult.items;
  const additionalModals = Array.isArray(menuResult) ? null : menuResult.modals;

  const CONTEXT_MENU_ITEMS = MENU_ITEMS.map(function CONTEXT_MENU_ITEMS(item) {
    return {
      ...item,
      action: () => {
        item.action();
      },
    };
  });

  return (
    <>
      {cycleDetails && (
        <div className="fixed">
          <CycleCreateUpdateModal
            data={cycleDetails}
            isOpen={updateModal}
            handleClose={() => setUpdateModal(false)}
            workspaceSlug={workspaceSlug}
            projectId={projectId}
          />
          <ArchiveCycleModal
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            cycleId={cycleId}
            isOpen={archiveCycleModal}
            handleClose={() => setArchiveCycleModal(false)}
          />
          <CycleDeleteModal
            cycle={cycleDetails}
            isOpen={deleteModal}
            handleClose={() => setDeleteModal(false)}
            workspaceSlug={workspaceSlug}
            projectId={projectId}
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
        maxHeight="lg"
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
