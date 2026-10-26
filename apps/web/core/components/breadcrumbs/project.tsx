/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRouter } from "@tanstack/react-router";

import { Logo } from "@plane/propel/emoji-icon-picker";
// plane imports
import type { TProject } from "@plane/types";
import { BreadcrumbNavigationSearchDropdown, Breadcrumbs } from "@plane/ui";
// hooks
import { getProjectSwitcherOptions, useProjectSwitcherData } from "@/components/navigation/project-switcher-options";

type TProjectBreadcrumbProps = {
  workspaceSlug: string;
  projectId: string;
  handleOnClick?: () => void;
};

export const ProjectBreadcrumb = function ProjectBreadcrumb(props: TProjectBreadcrumbProps) {
  const { workspaceSlug, projectId, handleOnClick } = props;
  // router
  const router = useRouter();
  // store hooks
  const {
    projectRows,
    joinedProjectIds,
    currentProject: currentProjectDetails,
  } = useProjectSwitcherData(workspaceSlug, projectId);

  // store hooks

  if (!currentProjectDetails) return null;

  // derived values
  const switcherOptions = getProjectSwitcherOptions(joinedProjectIds, projectRows);

  // helpers
  // oxlint-disable-next-line unicorn/consistent-function-scoping
  const renderIcon = (projectDetails: TProject) => (
    <span className="grid size-4 flex-shrink-0 place-items-center">
      <Logo logo={projectDetails.logo_props} size={14} />
    </span>
  );

  return (
    <>
      <Breadcrumbs.Item
        component={
          <BreadcrumbNavigationSearchDropdown
            selectedItem={currentProjectDetails.id}
            navigationItems={switcherOptions}
            onChange={(value: string) => {
              void router.navigate({ href: `/${workspaceSlug}/projects/${value}/issues` });
            }}
            title={currentProjectDetails?.name}
            icon={renderIcon(currentProjectDetails)}
            handleOnClick={() => {
              if (handleOnClick) handleOnClick();
              else void router.navigate({ href: `/${workspaceSlug}/projects/${currentProjectDetails.id}/issues/` });
            }}
            shouldTruncate
          />
        }
        showSeparator={false}
      />
    </>
  );
};
