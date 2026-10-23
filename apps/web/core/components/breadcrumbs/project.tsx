/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRouter } from "@tanstack/react-router";
import { observer } from "mobx-react";
import { Logo } from "@plane/propel/emoji-icon-picker";
import { ProjectIcon } from "@plane/propel/icons";
// plane imports
import type { ICustomSearchSelectOption } from "@plane/types";
import { BreadcrumbNavigationSearchDropdown, Breadcrumbs } from "@plane/ui";
import { SwitcherLabel } from "@/components/common/switcher-label";
// hooks
import { useProject } from "@/hooks/store/use-project";

import type { TProject } from "@plane/types";

type TProjectBreadcrumbProps = {
  workspaceSlug: string;
  projectId: string;
  handleOnClick?: () => void;
};

export const ProjectBreadcrumb = observer(function ProjectBreadcrumb(props: TProjectBreadcrumbProps) {
  const { workspaceSlug, projectId, handleOnClick } = props;
  // router
  const router = useRouter();
  // store hooks
  const { joinedProjectIds, getPartialProjectById } = useProject();
  const currentProjectDetails = getPartialProjectById(projectId);

  // store hooks

  if (!currentProjectDetails) return null;

  // derived values
  const switcherOptions = joinedProjectIds
    // oxlint-disable-next-line no-shadow
    .map((projectId) => {
      const project = getPartialProjectById(projectId);
      return {
        value: projectId,
        query: project?.name,
        content: (
          <SwitcherLabel
            name={project?.name}
            logo_props={project?.logo_props}
            LabelIcon={ProjectIcon}
            type="material"
          />
        ),
      };
    })
    .filter((option) => option !== undefined) as ICustomSearchSelectOption[];

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
});
