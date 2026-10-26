/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { EModalPosition, EModalWidth, ModalCore } from "@plane/ui";
// hooks
// local imports
import { InboxIssueCreateRoot } from "./create-root";

type TInboxIssueCreateModalRoot = {
  workspaceSlug: string;
  projectId: string;
  modalState: boolean;
  handleModalClose: () => void;
};

export function InboxIssueCreateModalRoot(props: TInboxIssueCreateModalRoot) {
  const { workspaceSlug, projectId, modalState, handleModalClose } = props;
  if (!modalState) return null;
  return (
    <ModalCore
      isOpen={modalState}
      position={EModalPosition.TOP}
      width={EModalWidth.XXXXL}
      className="rounded-lg !bg-transparent shadow-none transition-[width] ease-linear"
    >
      <InboxIssueCreateRoot
        key={`${workspaceSlug}/${projectId}`}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        handleModalClose={handleModalClose}
      />
    </ModalCore>
  );
}
