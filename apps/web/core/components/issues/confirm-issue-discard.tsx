/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMutation } from "@tanstack/react-query";
// ui
import { Button } from "@plane/propel/button";
import { EModalPosition, EModalWidth, ModalCore } from "@plane/ui";

type Props = {
  isOpen: boolean;
  handleClose: () => void;
  onDiscard: () => void;
  onConfirm: () => Promise<void>;
};

export function ConfirmIssueDiscard(props: Props) {
  const { isOpen, handleClose, onDiscard, onConfirm } = props;

  const save = useMutation({ mutationFn: onConfirm, retry: false });
  const onClose = () => {
    if (!save.isPending) handleClose();
  };

  return (
    <ModalCore isOpen={isOpen} handleClose={onClose} position={EModalPosition.TOP} width={EModalWidth.XXL}>
      <div className="px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
        <div className="sm:flex sm:items-start">
          <div className="mt-3 text-center sm:mt-0 sm:text-left">
            <h3 className="text-16 leading-6 font-medium text-primary">Save this draft?</h3>
            <div className="mt-2">
              <p className="text-13 text-secondary">
                You can save this work item to Drafts so you can come back to it later.{" "}
              </p>
            </div>
          </div>
        </div>
      </div>
      {save.isError && (
        <p role="alert" className="px-4 text-13 text-danger-primary">
          Could not save the draft. Please try again.
        </p>
      )}
      <div className="flex justify-between gap-2 p-4 sm:px-6">
        <div>
          <Button variant="secondary" onClick={onDiscard} disabled={save.isPending}>
            Discard
          </Button>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="secondary" onClick={onClose} disabled={save.isPending}>
            Cancel
          </Button>
          <Button variant="primary" onClick={() => save.mutate()} loading={save.isPending} disabled={save.isPending}>
            {save.isPending ? "Saving" : "Save to Drafts"}
          </Button>
        </div>
      </div>
    </ModalCore>
  );
}
