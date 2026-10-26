/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useParams } from "@tanstack/react-router";
import { useJoinedProjectIds } from "@/hooks/use-joined-project-ids";
import { useMemo, useState } from "react";
import type { ReactNode } from "react";

import type { ISearchIssueResponse } from "@plane/types";
import { IssueModalContext } from "./context";
import { useCreationPermissions } from "@/hooks/use-creation-permissions";

type Props = { allowedProjectIds?: string[]; children: ReactNode };
export const IssueModalProvider = function IssueModalProvider({ children, allowedProjectIds }: Props) {
  const [selectedParentIssue, setSelectedParentIssue] = useState<ISearchIssueResponse | null>(null);
  const { projectsWithCreatePermissions } = useCreationPermissions();
  const { workspaceSlug } = useParams({ strict: false });
  const joinedProjectIds = useJoinedProjectIds(workspaceSlug);
  const defaultProjectId = joinedProjectIds.find(
    (id) => projectsWithCreatePermissions[id] !== undefined && (!allowedProjectIds || allowedProjectIds.includes(id))
  );
  const value = useMemo(
    () => ({
      allowedProjectIds: allowedProjectIds ?? Object.keys(projectsWithCreatePermissions ?? {}),
      defaultProjectId,
      selectedParentIssue,
      setSelectedParentIssue,
    }),
    [allowedProjectIds, projectsWithCreatePermissions, defaultProjectId, selectedParentIssue]
  );
  return <IssueModalContext.Provider value={value}>{children}</IssueModalContext.Provider>;
};
