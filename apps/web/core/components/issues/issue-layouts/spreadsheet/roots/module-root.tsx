/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import React from "react";

// hooks
// components
import { ModuleIssueQuickActions } from "../../quick-action-dropdowns";
import { BaseSpreadsheetRoot } from "../base-spreadsheet-root";

export const ModuleSpreadsheetLayout = function ModuleSpreadsheetLayout() {
  const { moduleId } = useParams({ strict: false });

  return <BaseSpreadsheetRoot QuickActions={ModuleIssueQuickActions} viewId={moduleId?.toString()} />;
};
