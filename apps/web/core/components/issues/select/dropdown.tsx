/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import React from "react";

import type { IIssueLabel } from "@plane/types";
// hooks
import { useProjectLabelMutations } from "@/hooks/use-labels";
// local imports
import type { TWorkItemLabelSelectBaseProps } from "./base";
import { WorkItemLabelSelectBase } from "./base";

type TWorkItemLabelSelectProps = Omit<TWorkItemLabelSelectBaseProps, "labelIds" | "getLabelById"> & {
  projectId: string | undefined;
};

export function IssueLabelSelect(props: TWorkItemLabelSelectProps) {
  const { projectId } = props;
  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // store hooks
  const {
    labelIds: projectLabelIds,
    getLabelById,
    createLabel,
    canCreate,
    scope,
  } = useProjectLabelMutations(workspaceSlug?.toString(), projectId);
  // derived values
  const canCreateLabel = projectId && canCreate;

  const handleCreateLabel = (data: Partial<IIssueLabel>) => {
    if (!workspaceSlug || !projectId) {
      throw new Error("Workspace slug or project ID is missing");
    }
    return createLabel(data);
  };

  return (
    <WorkItemLabelSelectBase
      {...props}
      creationScope={JSON.stringify([scope.accountId, scope.sessionGeneration, workspaceSlug, projectId])}
      getLabelById={getLabelById}
      labelIds={projectLabelIds ?? []}
      createLabel={handleCreateLabel}
      createLabelEnabled={!!canCreateLabel}
    />
  );
}
