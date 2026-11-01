/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Header, Row } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { AppSidebarToggleButton } from "@/components/sidebar/sidebar-toggle-button";
import { useSidebarCollapsed } from "@/providers/sidebar-state";
import { useProjectNavigationPreferences } from "@/hooks/use-navigation-preferences";
import { TabNavigationRoot } from "@/components/navigation/tab-navigation-root";

type Props = {
  workspaceSlug: string;
  projectId: string;
  leftItemClassName?: string;
};

export function ProjectTabHeader({ workspaceSlug, projectId, leftItemClassName = "h-full max-w-full" }: Props) {
  const sidebarCollapsed = useSidebarCollapsed();
  const { preferences } = useProjectNavigationPreferences();
  if (preferences.navigationMode !== "TABBED") return null;

  return (
    <div className="z-20">
      <Row className="flex h-header w-full items-center gap-2 border-b border-subtle bg-surface-1">
        <div className="flex h-full w-full items-center gap-2 divide-x divide-subtle">
          <div className="flex size-full flex-1 items-center gap-2">
            {sidebarCollapsed && (
              <div className="shrink-0">
                <AppSidebarToggleButton />
              </div>
            )}
            <Header className={cn("h-full", { "pl-1.5": !sidebarCollapsed })}>
              <Header.LeftItem className={leftItemClassName}>
                <TabNavigationRoot workspaceSlug={workspaceSlug} projectId={projectId} />
              </Header.LeftItem>
            </Header>
          </div>
        </div>
      </Row>
    </div>
  );
}
