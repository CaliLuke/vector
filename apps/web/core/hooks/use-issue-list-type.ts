/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { EIssuesStoreType } from "@plane/types";
import { useLocation, useParams } from "@tanstack/react-router";

export const useIssueListType = () => {
  const { globalViewId, viewId, projectId, cycleId, moduleId, userId } = useParams({ strict: false });
  const pathname = useLocation({ select: (location) => location.pathname });

  if (globalViewId) return EIssuesStoreType.GLOBAL;

  if (userId) return EIssuesStoreType.PROFILE;

  if (viewId) return EIssuesStoreType.PROJECT_VIEW;

  if (cycleId) return EIssuesStoreType.CYCLE;

  if (moduleId) return EIssuesStoreType.MODULE;

  if (pathname.includes("/archives/")) return EIssuesStoreType.ARCHIVED;

  if (projectId) return EIssuesStoreType.PROJECT;

  return EIssuesStoreType.PROJECT;
};
