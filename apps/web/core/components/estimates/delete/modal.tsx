/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef } from "react";
import { Button } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { EModalPosition, EModalWidth, ModalCore } from "@plane/ui";
import { useEstimateEditorKey, useEstimateMutations } from "@/hooks/use-estimates";

type Props = {
  workspaceSlug: string;
  projectId: string;
  estimateId: string | undefined;
  isOpen: boolean;
  handleClose: () => void;
};

export function DeleteEstimateModal(props: Props) {
  const key = useEstimateEditorKey(props.workspaceSlug, props.projectId);
  return props.isOpen && props.estimateId ? (
    <DeleteEstimateConfirmation key={`${key}/${props.estimateId}`} {...props} estimateId={props.estimateId} />
  ) : null;
}

function DeleteEstimateConfirmation({
  workspaceSlug,
  projectId,
  estimateId,
  handleClose,
}: Props & { estimateId: string }) {
  const { estimates, mutation, isPending, isError, isAccessDenied, retry } = useEstimateMutations(
    workspaceSlug,
    projectId
  );
  const estimate = estimates?.find((row) => row.id === estimateId);
  const mounted = useRef(false);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  const remove = async () => {
    try {
      const result = await mutation.mutateAsync({ type: "delete", estimateId });
      if (!mounted.current || !result.captured.isCurrent()) return;
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Estimate deleted",
        message: "Estimate has been removed from your project.",
      });
      handleClose();
    } catch {
      if (mounted.current)
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Estimate deletion failed",
          message: "We were unable to delete the estimate. Please try again.",
        });
    }
  };
  const busy = mutation.isPending;
  const close = () => {
    if (!busy) handleClose();
  };
  return (
    <ModalCore isOpen handleClose={close} position={EModalPosition.TOP} width={EModalWidth.XXL}>
      <div className="relative space-y-6 py-5">
        <div className="relative flex items-center justify-between gap-2 px-5">
          <div className="text-18 font-medium text-primary">Delete Estimate System</div>
        </div>
        <div className="px-5">
          {isError || isAccessDenied ? (
            <div role="alert">
              Estimates are unavailable.{" "}
              <button
                type="button"
                onClick={() => {
                  void retry();
                }}
              >
                Retry
              </button>
            </div>
          ) : isPending ? (
            <div>Loading estimates…</div>
          ) : (
            <div className="text-14 text-secondary">
              Deleting the estimate <span className="font-bold text-primary">{estimate?.name}</span> system will remove
              it from all work items permanently. This action cannot be undone.
            </div>
          )}
        </div>
        <div className="relative flex items-center justify-end gap-3 border-t border-subtle px-5 pt-5">
          <Button variant="secondary" size="lg" onClick={close} disabled={busy}>
            Cancel
          </Button>
          <Button
            variant="error-fill"
            size="lg"
            onClick={remove}
            disabled={busy || isPending || isError || isAccessDenied || !estimate}
          >
            {busy ? "Deleting" : "Delete Estimate"}
          </Button>
        </div>
      </div>
    </ModalCore>
  );
}
