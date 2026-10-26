/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useLocation, useRouter } from "@tanstack/react-router";

// types
import { PROJECT_ERROR_MESSAGES } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { ICycle } from "@plane/types";
// ui
import { CatalogueDeleteConfirmation } from "@/components/cycles-modules/catalogue-delete-confirmation";
// hooks
import { useProjectCycles } from "@/hooks/use-cycles";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { isAuthorizationError } from "@/queries/authorization-errors";

interface ICycleDelete {
  cycle: ICycle;
  isOpen: boolean;
  handleClose: () => void;
  workspaceSlug: string;
  projectId: string;
}

export function CycleDeleteModal(props: ICycleDelete) {
  return props.isOpen ? (
    <CycleDeleteDialog key={JSON.stringify([props.workspaceSlug, props.projectId, props.cycle.id])} {...props} />
  ) : null;
}
function CycleDeleteDialog(props: ICycleDelete) {
  const { isOpen, handleClose, cycle, workspaceSlug, projectId } = props;
  // states
  // store hooks
  const { mutation, scope } = useProjectCycles(workspaceSlug, projectId);
  const mounted = useCommittedScope(
    JSON.stringify([scope.accountId, scope.sessionGeneration, workspaceSlug, projectId, cycle.id, isOpen])
  );
  const { t } = useTranslation();
  // router
  const router = useRouter();
  const { cycleId } = useParams({ strict: false });
  const searchParams = new URLSearchParams(useLocation({ select: (location) => location.searchStr }));
  const peekCycle = searchParams.get("peekCycle");

  const formSubmit = async () => {
    if (!mounted()) return;
    try {
      const result = await mutation.mutateAsync({ type: "delete", id: cycle.id });
      if (!mounted() || !result.captured.isCurrent()) return;
      if (cycleId === cycle.id || peekCycle === cycle.id)
        void router.navigate({ href: `/${workspaceSlug}/projects/${projectId}/cycles` });
      setToast({ type: TOAST_TYPE.SUCCESS, title: "Success!", message: "Cycle deleted successfully." });
      handleClose();
    } catch (error) {
      if (!mounted()) return;
      const message = isAuthorizationError(error)
        ? PROJECT_ERROR_MESSAGES.permissionError
        : PROJECT_ERROR_MESSAGES.cycleDeleteError;
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t(message.i18n_title),
        message: message.i18n_message && t(message.i18n_message),
      });
    }
  };

  return (
    <CatalogueDeleteConfirmation
      entity="cycle"
      itemName={cycle?.name}
      handleClose={handleClose}
      handleSubmit={formSubmit}
      isSubmitting={mutation.isPending}
      isOpen={isOpen}
    />
  );
}
