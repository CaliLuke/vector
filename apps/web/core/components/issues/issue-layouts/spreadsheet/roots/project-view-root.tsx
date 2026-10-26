/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import React from "react";

// hooks
// components
import { ProjectIssueQuickActions } from "../../quick-action-dropdowns";
import { BaseSpreadsheetRoot } from "../base-spreadsheet-root";
// types
// constants

export const ProjectViewSpreadsheetLayout = function ProjectViewSpreadsheetLayout() {
  const { viewId } = useParams({ from: "/_authenticated/$workspaceSlug/_projects/projects/$projectId/views/$viewId" });

  return <BaseSpreadsheetRoot QuickActions={ProjectIssueQuickActions} viewId={viewId.toString()} />;
};
