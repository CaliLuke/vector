/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import React from "react";

// hooks
// constant
// types
import { ProjectIssueQuickActions } from "../../quick-action-dropdowns";
// components
import { BaseKanBanRoot } from "../base-kanban-root";

export const ProjectViewKanBanLayout = function ProjectViewKanBanLayout() {
  const { viewId } = useParams({ from: "/_authenticated/$workspaceSlug/_projects/projects/$projectId/views/$viewId" });

  return <BaseKanBanRoot QuickActions={ProjectIssueQuickActions} viewId={viewId.toString()} />;
};
