/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link, useParams, useRouter } from "@tanstack/react-router";

// ui
import { MODULE_STATUS } from "@plane/constants";
import { ModuleStatusIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
// components
import { SIDEBAR_WIDTH } from "@/components/gantt-chart/constants";
import { getBlockViewDetails } from "@/components/issues/issue-layouts/utils";
// constants
// hooks
import type { IModule } from "@plane/types";

import { usePlatformOS } from "@/hooks/use-platform-os";

type Props = {
  module: IModule;
};

export const ModuleGanttBlock = function ModuleGanttBlock(props: Props) {
  const { module: moduleDetails } = props;
  // router
  const router = useRouter();
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // store hooks
  // derived values
  // hooks
  const { isMobile } = usePlatformOS();

  const { message, blockStyle } = getBlockViewDetails(
    moduleDetails,
    MODULE_STATUS.find((s) => s.value === moduleDetails?.status)?.color ?? ""
  );

  if (!moduleDetails) return null;

  return (
    <Tooltip
      isMobile={isMobile}
      tooltipContent={
        <div className="space-y-1">
          <h5>{moduleDetails?.name}</h5>
          <div>{message}</div>
        </div>
      }
      position="top-start"
    >
      <button
        type="button"
        aria-label={`Open module ${moduleDetails?.name ?? ""}`}
        className="relative flex h-full w-full cursor-pointer items-center rounded-sm text-left"
        style={blockStyle}
        onClick={() =>
          void router.navigate({
            href: `/${workspaceSlug}/projects/${moduleDetails?.project_id}/modules/${moduleDetails?.id}`,
          })
        }
      >
        <div className="absolute top-0 left-0 h-full w-full bg-surface-1/50" />
        <div
          className="sticky w-auto truncate overflow-hidden px-2.5 py-1 text-13 text-primary"
          style={{ left: `${SIDEBAR_WIDTH}px` }}
        >
          {moduleDetails?.name}
        </div>
      </button>
    </Tooltip>
  );
};

export const ModuleGanttSidebarBlock = function ModuleGanttSidebarBlock(props: Props) {
  const { module: moduleDetails } = props;
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // store hooks
  // derived values
  if (!moduleDetails) return null;

  return (
    <Link
      className="relative flex h-full w-full items-center gap-2"
      to="/$workspaceSlug/projects/$projectId/modules/$moduleId/"
      params={{
        workspaceSlug,
        projectId: moduleDetails.project_id,
        moduleId: moduleDetails.id,
      }}
      draggable={false}
    >
      <ModuleStatusIcon status={moduleDetails?.status ?? "backlog"} height="16px" width="16px" />
      <h6 className="flex-grow truncate text-13 font-medium">{moduleDetails?.name}</h6>
    </Link>
  );
};
