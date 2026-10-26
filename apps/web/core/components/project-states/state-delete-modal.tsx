/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// Vector imports
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IState } from "@plane/types";
// ui
import { AlertModalCore } from "@plane/ui";
// hooks
import { useProjectStates } from "@/hooks/use-project-states";

type TStateDeleteModal = {
  isOpen: boolean;
  onClose: () => void;
  data: IState | null;
};

export function StateDeleteModal(props: TStateDeleteModal) {
  const { isOpen, onClose, data } = props;
  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const { deleteState, mutation } = useProjectStates(workspaceSlug, data?.project_id);

  const handleClose = () => {
    onClose();
  };

  const handleDeletion = async () => {
    if (!workspaceSlug || !data) return;

    await deleteState(data.id)
      .then(() => {
        handleClose();
      })
      .catch((err) => {
        if (err.status === 400)
          setToast({
            type: TOAST_TYPE.ERROR,
            title: "Error!",
            message:
              "This state contains some work items within it, please move them to some other state to delete this state.",
          });
        else
          setToast({
            type: TOAST_TYPE.ERROR,
            title: "Error!",
            message: "State could not be deleted. Please try again.",
          });
      });
  };

  return (
    <AlertModalCore
      handleClose={handleClose}
      handleSubmit={handleDeletion}
      isSubmitting={mutation.isPending}
      isOpen={isOpen}
      title="Delete State"
      content={
        <>
          Are you sure you want to delete state- <span className="font-medium text-primary">{data?.name}</span>? All of
          the data related to the state will be permanently removed. This action cannot be undone.
        </>
      }
    />
  );
}
