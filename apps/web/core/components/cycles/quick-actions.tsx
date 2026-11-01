/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRouter } from "@tanstack/react-router";

// ui
import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { QuickActionsMenu } from "@/components/common/quick-actions-menu";
import { normalizeQuickActions, useQuickActionDialogs } from "@/components/common/quick-actions-helper";
import { copyUrlToClipboard } from "@plane/utils/string";
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
  const dialogs = useQuickActionDialogs();
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
    copyUrlToClipboard(cycleLink)
      .then(() => {
        if (!mounted()) return;
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("common.link_copied"),
          message: t("common.link_copied_to_clipboard"),
        });
        return;
      })
      .catch(() => {
        if (!mounted()) return;
        setToast({ type: TOAST_TYPE.ERROR, title: "Could not copy to clipboard", message: "Please try again." });
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
    ...dialogs.handlers,
    handleRestore: () => void handleRestoreCycle(),
    handleCopyLink: () => void handleCopyText(),
    handleOpenInNewTab,
  });

  const { items: MENU_ITEMS, modals: additionalModals } = normalizeQuickActions(menuResult);

  return (
    <>
      {cycleDetails && (
        <div className="fixed">
          <CycleCreateUpdateModal
            data={cycleDetails}
            isOpen={dialogs.edit.isOpen}
            handleClose={dialogs.edit.close}
            workspaceSlug={workspaceSlug}
            projectId={projectId}
          />
          <ArchiveCycleModal
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            cycleId={cycleId}
            isOpen={dialogs.archive.isOpen}
            handleClose={dialogs.archive.close}
          />
          <CycleDeleteModal
            cycle={cycleDetails}
            isOpen={dialogs.delete.isOpen}
            handleClose={dialogs.delete.close}
            workspaceSlug={workspaceSlug}
            projectId={projectId}
          />
          {additionalModals}
        </div>
      )}
      <QuickActionsMenu parentRef={parentRef} items={MENU_ITEMS} className={customClassName} maxHeight="lg" />
    </>
  );
};
