/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TIssueServiceType } from "@plane/types";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { LinkModal } from "@/components/common/link-modal";
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import type { TLinkOperations } from "@/hooks/use-issue-link-operations";

type TLinkOperationsModal = Pick<TLinkOperations, "create" | "update" | "isCurrent">;

export type TIssueLinkCreateEditModal = {
  isModalOpen: boolean;
  handleOnClose?: () => void;
  linkOperations: TLinkOperationsModal;
  issueServiceType: TIssueServiceType;
};

export const IssueLinkCreateUpdateModal = function IssueLinkCreateUpdateModal({
  isModalOpen,
  handleOnClose,
  linkOperations,
  issueServiceType,
}: TIssueLinkCreateEditModal) {
  const { issueLinkData, setIssueLinkData } = useIssueDetailUiState(issueServiceType);
  const isModalCurrent = useCommittedScope(JSON.stringify([isModalOpen, issueLinkData?.id, issueServiceType]));
  return (
    <LinkModal
      isOpen={isModalOpen}
      link={issueLinkData}
      onClose={() => {
        if (!isModalCurrent() || !linkOperations.isCurrent()) return;
        setIssueLinkData(null);
        handleOnClose?.();
      }}
      onSave={(value) =>
        issueLinkData?.id ? linkOperations.update(issueLinkData.id, value) : linkOperations.create(value)
      }
    />
  );
};
