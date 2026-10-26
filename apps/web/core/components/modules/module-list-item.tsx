/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useLocation, useRouter } from "@tanstack/react-router";
import React, { useRef } from "react";

// icons
import type { IModule } from "@plane/types";
import { Info } from "lucide-react";
import { CheckIcon } from "@plane/propel/icons";
// ui
import { CircularProgressIndicator } from "@plane/ui";
// components
import { ListItem } from "@/components/core/list";
import { ModuleListItemAction, ModuleQuickActions } from "@/components/modules";
// helpers
// hooks

import { usePlatformOS } from "@/hooks/use-platform-os";
import { getModulePeekHref } from "./module-peek-navigation";

type Props = {
  module: IModule;
};

export const ModuleListItem = function ModuleListItem(props: Props) {
  const { module: moduleDetails } = props;
  const moduleId = moduleDetails.id;
  // refs
  const parentRef = useRef(null);
  // router
  const router = useRouter();
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const projectId = moduleDetails.project_id;
  const searchParams = new URLSearchParams(useLocation({ select: (location) => location.searchStr }));
  const pathname = useLocation({ select: (location) => location.pathname });
  // store hooks
  const { isMobile } = usePlatformOS();

  // derived values

  if (!moduleDetails) return null;

  const completionPercentage =
    ((moduleDetails.completed_issues + moduleDetails.cancelled_issues) / moduleDetails.total_issues) * 100;

  const progress = isNaN(completionPercentage) ? 0 : Math.floor(completionPercentage);

  const completedModuleCheck = moduleDetails.status === "completed";

  // handlers
  const openModuleOverview = (e: React.MouseEvent<HTMLButtonElement | HTMLAnchorElement>) => {
    e.stopPropagation();
    e.preventDefault();

    void router.navigate({ href: getModulePeekHref(pathname, searchParams.toString(), moduleId) });
  };

  const handleArchivedModuleClick = (e: React.MouseEvent<HTMLButtonElement | HTMLAnchorElement>) => {
    openModuleOverview(e);
  };

  const handleItemClick = moduleDetails.archived_at ? handleArchivedModuleClick : undefined;

  return (
    <ListItem
      title={moduleDetails?.name ?? ""}
      itemLink={`/${workspaceSlug?.toString()}/projects/${moduleDetails.project_id}/modules/${moduleDetails.id}`}
      onItemClick={handleItemClick}
      prependTitleElement={
        <CircularProgressIndicator size={30} percentage={progress} strokeWidth={3}>
          {completedModuleCheck ? (
            progress === 100 ? (
              <CheckIcon className="h-3 w-3 stroke-[2] text-accent-primary" />
            ) : (
              <span className="text-13 text-accent-primary">{`!`}</span>
            )
          ) : progress === 100 ? (
            <CheckIcon className="h-3 w-3 stroke-[2] text-accent-primary" />
          ) : (
            <span className="text-9 text-tertiary">{`${progress}%`}</span>
          )}
        </CircularProgressIndicator>
      }
      appendTitleElement={
        <button
          onClick={openModuleOverview}
          className={`z-[5] flex-shrink-0 ${isMobile ? "flex" : "hidden group-hover:flex"}`}
        >
          <Info className="h-4 w-4 text-placeholder" />
        </button>
      }
      actionableItems={<ModuleListItemAction moduleId={moduleId} moduleDetails={moduleDetails} parentRef={parentRef} />}
      quickActionElement={
        <div className="block md:hidden">
          <ModuleQuickActions
            module={moduleDetails}
            parentRef={parentRef}
            moduleId={moduleId}
            projectId={projectId}
            workspaceSlug={workspaceSlug}
          />
        </div>
      }
      isMobile={isMobile}
      parentRef={parentRef}
    />
  );
};
