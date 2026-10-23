/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useContext } from "react";

import { EIssuesStoreType } from "@plane/types";
import { useIssues } from "./store/use-issues";
import { useParams } from "@tanstack/react-router";

export const IssuesStoreContext = createContext<EIssuesStoreType | undefined>(undefined);

export const useIssueStoreType = () => {
  const storeType = useContext(IssuesStoreContext);
  const { globalViewId, viewId, projectId, cycleId, moduleId, userId } = useParams({ strict: false });

  // If store type exists in context, use that store type
  if (storeType) return storeType;

  // else check the router params to determine the issue store
  if (globalViewId) return EIssuesStoreType.GLOBAL;

  if (userId) return EIssuesStoreType.PROFILE;

  if (viewId) return EIssuesStoreType.PROJECT_VIEW;

  if (cycleId) return EIssuesStoreType.CYCLE;

  if (moduleId) return EIssuesStoreType.MODULE;

  if (projectId) return EIssuesStoreType.PROJECT;

  return EIssuesStoreType.PROJECT;
};

export const useIssuesStore = () => {
  const storeType = useIssueStoreType();

  return useIssues(storeType);
};
