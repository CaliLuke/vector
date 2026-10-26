/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// types
import { PROJECT_ERROR_MESSAGES } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IModule } from "@plane/types";
// ui
import { CatalogueDeleteConfirmation } from "@/components/cycles-modules/catalogue-delete-confirmation";
// constants
// hooks
import { useProjectModules } from "@/hooks/use-modules";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { isAuthorizationError } from "@/queries/authorization-errors";

type Props = {
  data: IModule;
  isOpen: boolean;
  onClose: () => void;
  onDeleteSuccess: (module: IModule) => void;
};

export function DeleteModuleModal(props: Props) {
  return props.isOpen ? <ModuleDeleteDialog key={props.data.id} {...props} /> : null;
}
function ModuleDeleteDialog(props: Props) {
  const { data, isOpen, onClose, onDeleteSuccess } = props;
  // states
  // router
  const { workspaceSlug, projectId } = useParams({ strict: false });
  // store hooks
  const { mutation, scope } = useProjectModules(workspaceSlug, projectId);
  const mounted = useCommittedScope(
    JSON.stringify([scope.accountId, scope.sessionGeneration, workspaceSlug, projectId, data.id, isOpen])
  );
  const { t } = useTranslation();

  const handleClose = () => onClose();
  const handleDeletion = async () => {
    if (!mounted() || !workspaceSlug || !projectId) return;
    try {
      const result = await mutation.mutateAsync({ type: "delete", id: data.id });
      if (!mounted() || !result.captured.isCurrent()) return;
      onDeleteSuccess(data);
      onClose();
      setToast({ type: TOAST_TYPE.SUCCESS, title: "Success!", message: "Module deleted successfully." });
    } catch (error) {
      if (!mounted()) return;
      const message = isAuthorizationError(error)
        ? PROJECT_ERROR_MESSAGES.permissionError
        : PROJECT_ERROR_MESSAGES.moduleDeleteError;
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t(message.i18n_title),
        message: message.i18n_message && t(message.i18n_message),
      });
    }
  };

  return (
    <CatalogueDeleteConfirmation
      entity="module"
      itemName={data?.name}
      handleClose={handleClose}
      handleSubmit={handleDeletion}
      isSubmitting={mutation.isPending}
      isOpen={isOpen}
    />
  );
}
