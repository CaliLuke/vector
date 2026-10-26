/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef } from "react";

// types
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IIssueLabel } from "@plane/types";
// ui
import { AlertModalCore } from "@plane/ui";
// hooks
import { useProjectLabelMutations } from "@/hooks/use-labels";

type Props = {
  workspaceSlug: string;
  projectId: string;
  onClose: () => void;
  data: IIssueLabel;
};

export function DeleteLabelModal(props: Props) {
  const { workspaceSlug, projectId, onClose, data } = props;
  const { deleteLabel, mutation } = useProjectLabelMutations(workspaceSlug, projectId);
  const mounted = useRef(false);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  const handleDeletion = async () => {
    try {
      await deleteLabel(data.id);
      if (mounted.current) onClose();
    } catch (err) {
      if (!mounted.current) return;
      const error =
        err && typeof err === "object" && "error" in err && typeof err.error === "string"
          ? err.error
          : "Label could not be deleted. Please try again.";
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: error,
      });
    }
  };

  return (
    <AlertModalCore
      handleClose={onClose}
      handleSubmit={handleDeletion}
      isSubmitting={mutation.isPending}
      isOpen
      title="Delete Label"
      content={
        <>
          Are you sure you want to delete <span className="font-medium text-primary">{data.name}</span>? This will
          remove the label from all the work item and from any views where the label is being filtered upon.
        </>
      }
    />
  );
}
