/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useState } from "react";

import { PlusIcon } from "@plane/propel/icons";
// plane imports
import type { TIssueLink } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
// hooks
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { useLinkOperations } from "@/hooks/use-issue-link-operations";
// local imports
import { IssueLinkCreateUpdateModal } from "./create-update-link-modal";
import { IssueLinkList } from "./links";

export type TLinkOperations = {
  create: (data: Partial<TIssueLink>) => Promise<void>;
  update: (linkId: string, data: Partial<TIssueLink>) => Promise<void>;
  remove: (linkId: string) => Promise<void>;
};

export type TIssueLinkRoot = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  disabled?: boolean;
};

export function IssueLinkRoot(props: TIssueLinkRoot) {
  // props
  const { workspaceSlug, projectId, issueId, disabled = false } = props;
  // hooks
  const { toggleIssueLinkModal: toggleIssueLinkModalStore } = useIssueDetailUiState();
  // state
  const [isIssueLinkModal, setIsIssueLinkModal] = useState(false);
  const toggleIssueLinkModal = useCallback(
    (modalToggle: boolean) => {
      toggleIssueLinkModalStore(modalToggle);
      setIsIssueLinkModal(modalToggle);
    },
    [toggleIssueLinkModalStore]
  );

  const handleOnClose = useCallback(() => toggleIssueLinkModal(false), [toggleIssueLinkModal]);
  const handleLinkOperations = useLinkOperations(
    workspaceSlug,
    projectId,
    issueId,
    EIssueServiceType.ISSUES,
    handleOnClose
  );

  return (
    <>
      <IssueLinkCreateUpdateModal
        isModalOpen={isIssueLinkModal}
        handleOnClose={handleOnClose}
        linkOperations={handleLinkOperations}
        issueServiceType={EIssueServiceType.ISSUES}
      />

      <div className="py-1 text-11">
        <div className="flex items-center justify-between gap-2">
          <h4>Links</h4>
          {!disabled && (
            <button
              type="button"
              className={`grid h-7 w-7 place-items-center rounded-sm p-1 duration-300 outline-none hover:bg-surface-2 ${
                disabled ? "cursor-not-allowed" : "cursor-pointer"
              }`}
              onClick={() => toggleIssueLinkModal(true)}
              disabled={disabled}
            >
              <PlusIcon className="h-4 w-4" />
            </button>
          )}
        </div>

        <div>
          <IssueLinkList
            issueId={issueId}
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            linkOperations={handleLinkOperations}
            disabled={disabled}
          />
        </div>
      </div>
    </>
  );
}
