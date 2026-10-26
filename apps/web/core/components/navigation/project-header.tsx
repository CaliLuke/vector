/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRouter } from "@tanstack/react-router";
import { useCallback, useMemo } from "react";

// plane imports
import { CustomSearchSelect } from "@plane/ui";
// hooks

// local imports
import { ProjectHeaderButton } from "./project-header-button";
import { getTabUrl } from "./tab-navigation-utils";
import { useTabPreferences } from "./use-tab-preferences";
import { useNavigationItems } from "./use-navigation-items";
import { getProjectSwitcherOptions, useProjectSwitcherData } from "./project-switcher-options";

type TProjectHeaderProps = {
  workspaceSlug: string;
  projectId: string;
};

export const ProjectHeader = function ProjectHeader(props: TProjectHeaderProps) {
  const { workspaceSlug, projectId } = props;
  // router
  const router = useRouter();
  // store hooks
  const {
    projectRows,
    joinedProjectIds,
    currentProject: currentProjectDetails,
  } = useProjectSwitcherData(workspaceSlug, projectId);

  // Get current project details
  // Get available navigation items for this project
  const navigationItems = useNavigationItems({ workspaceSlug, projectId, project: currentProjectDetails });

  // Get preferences from hook
  const { tabPreferences } = useTabPreferences(workspaceSlug, projectId);

  // Memoize available tab keys
  const availableTabKeys = useMemo(() => navigationItems.map((item) => item.key), [navigationItems]);

  // Memoize validated default tab key
  const validatedDefaultTabKey = useMemo(
    () =>
      availableTabKeys.includes(tabPreferences.defaultTab)
        ? tabPreferences.defaultTab
        : availableTabKeys[0] || "work_items",
    [availableTabKeys, tabPreferences.defaultTab]
  );

  // Memoize switcher options to prevent recalculation on every render
  const switcherOptions = useMemo(
    () => getProjectSwitcherOptions(joinedProjectIds, projectRows),
    [joinedProjectIds, projectRows]
  );

  // Memoize onChange handler
  const handleProjectChange = useCallback(
    (value: string) => {
      if (value !== currentProjectDetails?.id) {
        void router.navigate({ href: getTabUrl(workspaceSlug, value, validatedDefaultTabKey) });
      }
    },
    [currentProjectDetails?.id, router, workspaceSlug, validatedDefaultTabKey]
  );

  // Early return if no project details
  if (!currentProjectDetails) return null;

  return (
    <CustomSearchSelect
      options={switcherOptions}
      value={currentProjectDetails.id}
      onChange={handleProjectChange}
      customButton={currentProjectDetails ? <ProjectHeaderButton project={currentProjectDetails} /> : null}
      className="h-full rounded"
      customButtonClassName="group flex items-center gap-0.5 rounded-sm hover:bg-surface-2 outline-none cursor-pointer h-full"
    />
  );
};
