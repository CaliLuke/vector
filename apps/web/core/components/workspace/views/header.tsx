/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";

// plane imports
import { DEFAULT_GLOBAL_VIEWS_LIST, EUserPermissions } from "@plane/constants";
import { PlusIcon } from "@plane/propel/icons";
import type { IWorkspaceView, TStaticViewTypes } from "@plane/types";
import { Header, EHeaderVariant } from "@plane/ui";
// hooks
import { useWorkspacePermissions } from "@/hooks/use-permissions";
import { useWorkspaceViews } from "@/hooks/use-workspace-views";
// local imports
import { DefaultWorkspaceViewQuickActions } from "./default-view-quick-action";
import { CreateUpdateWorkspaceViewModal } from "./modal";
import { WorkspaceViewQuickActions } from "./quick-action";

function ViewTab(props: { view: IWorkspaceView }) {
  const { view } = props;
  // refs
  const parentRef = useRef<HTMLDivElement>(null);
  // router
  const { workspaceSlug, globalViewId } = useParams({ strict: false });
  if (!workspaceSlug || !globalViewId) return null;

  return (
    <div ref={parentRef} className="relative">
      <WorkspaceViewQuickActions workspaceSlug={workspaceSlug?.toString()} view={view} />
    </div>
  );
}

function DefaultViewTab(props: {
  tab: {
    key: TStaticViewTypes;
    i18n_label: string;
  };
}) {
  const { tab } = props;
  // refs
  const parentRef = useRef<HTMLDivElement>(null);
  // router
  const { workspaceSlug, globalViewId } = useParams({ strict: false });

  if (!workspaceSlug || !globalViewId) return null;
  return (
    <div key={tab.key} ref={parentRef} className="relative">
      <DefaultWorkspaceViewQuickActions workspaceSlug={workspaceSlug?.toString()} view={tab} />
    </div>
  );
}

export function GlobalViewsHeader() {
  // states
  const [createViewModal, setCreateViewModal] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  // router
  const { workspaceSlug, globalViewId } = useParams({ strict: false });
  const { views: currentWorkspaceViews } = useWorkspaceViews(workspaceSlug);
  const { can: canWorkspace } = useWorkspacePermissions(workspaceSlug);

  // bring the active view to the centre of the header
  useEffect(() => {
    if (globalViewId && currentWorkspaceViews) {
      const activeTabElement = document.querySelector(`#global-view-${globalViewId.toString()}`);
      if (activeTabElement && containerRef.current) {
        const containerRect = containerRef.current.getBoundingClientRect();
        const activeTabRect = activeTabElement.getBoundingClientRect();
        const diff = containerRect.right - activeTabRect.right;
        activeTabElement.scrollIntoView({ behavior: "smooth", inline: diff > 500 ? "center" : "nearest" });
      }
    }
  }, [globalViewId, currentWorkspaceViews, containerRef]);

  const isAuthorizedUser = canWorkspace([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  return (
    <Header variant={EHeaderVariant.SECONDARY} className="z-[12] min-h-[44px] bg-surface-1">
      <CreateUpdateWorkspaceViewModal isOpen={createViewModal} onClose={() => setCreateViewModal(false)} />
      <div
        ref={containerRef}
        className="horizontal-scrollbar flex scrollbar-sm h-full w-full items-center overflow-x-auto overflow-y-hidden"
      >
        {DEFAULT_GLOBAL_VIEWS_LIST.map((tab) => (
          <DefaultViewTab key={tab.key} tab={tab} />
        ))}

        {currentWorkspaceViews?.map((view) => (
          <ViewTab key={view.id} view={view} />
        ))}
      </div>

      {isAuthorizedUser ? (
        <button
          type="button"
          className="sticky -right-4 flex flex-shrink-0 items-center justify-center border-transparent bg-surface-1 py-3 hover:border-subtle hover:text-placeholder"
          onClick={() => setCreateViewModal(true)}
        >
          <PlusIcon className="h-4 w-4 text-accent-secondary" />
        </button>
      ) : (
        <></>
      )}
    </Header>
  );
}
