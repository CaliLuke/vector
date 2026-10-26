/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useWorkspaceEstimates } from "./use-estimates";
import { useWorkspaceCycles } from "./use-cycles";
import { useWorkspaceModules } from "./use-modules";
import { useWorkspaceLabels } from "./use-labels";
export function useWorkspaceIssueProperties(workspaceSlug: string | undefined) {
  useWorkspaceLabels(workspaceSlug);
  useWorkspaceEstimates(workspaceSlug);
  useWorkspaceCycles(workspaceSlug);
  useWorkspaceModules(workspaceSlug);
}
