/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { useParams } from "@tanstack/react-router";
import type { ISearchIssueResponse, TIssue } from "@plane/types";
import { ExistingIssuesListModal } from "@/components/core/modals/existing-issues-list-modal";
import { CreateUpdateEpicModal } from "@/components/epic-modal";
import { CreateUpdateIssueModal } from "@/components/issues/issue-modal/modal";
import { useIssueListType } from "@/hooks/use-issue-list-type";
import { addSelectedIssuesToView } from "./add-selected-issues";

type Props = {
  issuePayload: Partial<TIssue>;
  isEpic?: boolean;
  addIssuesToView?: (issueIds: string[]) => Promise<void | TIssue>;
};

export function useGroupWorkItemModals({ issuePayload, isEpic = false, addIssuesToView }: Props) {
  const [createOpen, setCreateOpen] = useState(false);
  const [existingOpen, setExistingOpen] = useState(false);
  const { workspaceSlug, projectId, moduleId, cycleId } = useParams({ strict: false });
  const storeType = useIssueListType();
  const canAddExisting = Boolean(moduleId || cycleId);
  const addExisting = async (data: ISearchIssueResponse[]) => {
    if (!workspaceSlug || !projectId) return;
    await addSelectedIssuesToView(data, addIssuesToView);
  };

  return {
    canAddExisting,
    openCreate: () => setCreateOpen(true),
    openExisting: () => setExistingOpen(true),
    modals: (
      <>
        {isEpic ? (
          <CreateUpdateEpicModal isOpen={createOpen} onClose={() => setCreateOpen(false)} data={issuePayload} />
        ) : (
          <CreateUpdateIssueModal
            isOpen={createOpen}
            onClose={() => setCreateOpen(false)}
            data={issuePayload}
            storeType={storeType}
          />
        )}
        {canAddExisting && (
          <ExistingIssuesListModal
            workspaceSlug={workspaceSlug?.toString()}
            projectId={projectId?.toString()}
            isOpen={existingOpen}
            handleClose={() => setExistingOpen(false)}
            searchParams={moduleId ? { module: moduleId } : { cycle: true }}
            handleOnSubmit={addExisting}
          />
        )}
      </>
    ),
  };
}
