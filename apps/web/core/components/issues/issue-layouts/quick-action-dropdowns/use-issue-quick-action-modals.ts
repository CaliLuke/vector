/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useState } from "react";
import type { TIssue } from "@plane/types";

export const useIssueQuickActionModals = (issue: TIssue) => {
  const { id: sourceIssueId, ...copyProperties } = issue;
  const [createUpdateIssueModal, setCreateUpdateIssueModal] = useState(false);
  const [issueToEdit, setIssueToEdit] = useState<TIssue | undefined>(undefined);
  const [deleteIssueModal, setDeleteIssueModal] = useState(false);
  const [archiveIssueModal, setArchiveIssueModal] = useState(false);
  const closeCreateUpdateIssueModal = useCallback(() => {
    setCreateUpdateIssueModal(false);
    setIssueToEdit(undefined);
  }, []);

  return {
    data: issueToEdit ?? { ...copyProperties, name: `${issue.name} (copy)`, sourceIssueId },
    createUpdateIssueModal,
    setCreateUpdateIssueModal,
    closeCreateUpdateIssueModal,
    issueToEdit,
    setIssueToEdit,
    deleteIssueModal,
    setDeleteIssueModal,
    archiveIssueModal,
    setArchiveIssueModal,
  };
};
