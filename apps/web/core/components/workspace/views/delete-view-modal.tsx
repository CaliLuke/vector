/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// Vector Imports
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IWorkspaceView } from "@plane/types";
import { AlertModalCore } from "@plane/ui";
// hooks
import { useWorkspaceViewActionScope, useWorkspaceViewMutations } from "@/hooks/use-workspace-views";

type Props = {
  data: IWorkspaceView;
  isOpen: boolean;
  onClose: () => void;
};

export function DeleteGlobalViewModal(props: Props) {
  const { data, isOpen, onClose } = props;
  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const { remove } = useWorkspaceViewMutations(workspaceSlug);
  const captureActionScope = useWorkspaceViewActionScope(workspaceSlug, isOpen, data.id);
  const { t } = useTranslation();
  const handleClose = () => onClose();

  const handleDeletion = async () => {
    if (!workspaceSlug) return;
    const isCurrent = captureActionScope();
    try {
      await remove.mutateAsync(data.id);
      if (!isCurrent()) return;
      handleClose();
    } catch (_error) {
      if (!isCurrent()) return;
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Failed to delete the view. Please try again.",
      });
      return;
    }
  };

  return (
    <AlertModalCore
      handleClose={handleClose}
      handleSubmit={handleDeletion}
      isSubmitting={remove.isPending}
      isOpen={isOpen}
      title={t("workspace_views.delete_view.title")}
      content={<>{t("workspace_views.delete_view.content")}</>}
    />
  );
}
