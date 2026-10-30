/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { Breadcrumbs } from "@plane/ui";
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";

// local components
import { useProjectNavigationPreferences } from "@/hooks/use-navigation-preferences";
import { ProjectBreadcrumb } from "./project";

type TCommonProjectBreadcrumbProps = {
  workspaceSlug: string;
  projectId: string;
};

export function CommonProjectBreadcrumbs(props: TCommonProjectBreadcrumbProps) {
  const { workspaceSlug, projectId } = props;
  // preferences
  const { preferences: projectPreferences } = useProjectNavigationPreferences();

  if (projectPreferences.navigationMode === "TABBED") return null;
  return <ProjectBreadcrumb workspaceSlug={workspaceSlug} projectId={projectId} />;
}

// Own the breadcrumb container so its children retain their responsive ordering.
export function ProjectSectionBreadcrumbs({
  workspaceSlug,
  projectId,
  onBack,
  isLoading,
  label,
  href,
  icon,
  sectionIsLast,
  children,
}: TCommonProjectBreadcrumbProps & {
  onBack: () => void;
  isLoading: boolean;
  label: string;
  href: string;
  icon: ReactNode;
  sectionIsLast?: boolean;
  children: ReactNode;
}) {
  return (
    <Breadcrumbs onBack={onBack} isLoading={isLoading}>
      <CommonProjectBreadcrumbs workspaceSlug={workspaceSlug} projectId={projectId} />
      <Breadcrumbs.Item
        component={<BreadcrumbLink label={label} href={href} icon={icon} isLast={sectionIsLast} />}
        isLast={sectionIsLast}
      />
      {children}
    </Breadcrumbs>
  );
}
