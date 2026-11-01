/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { AnalyticsSidebarLoader } from "@/components/common/analytics-sidebar-loader";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";

// plane imports

// local imports
import useCyclesDetails from "../active-cycle/use-cycles-details";
import { CycleAnalyticsProgress } from "./issue-progress";
import { CycleSidebarDetails } from "./sidebar-details";
import { CycleSidebarHeader } from "./sidebar-header";

type Props = {
  handleClose: () => void;
  isArchived?: boolean;
  cycleId: string;
  projectId: string;
  workspaceSlug: string;
};

export const CycleDetailsSidebar = function CycleDetailsSidebar(props: Props) {
  const { handleClose, isArchived, projectId, workspaceSlug, cycleId } = props;

  // store hooks
  const {
    cycle: cycleDetails,
    isError,
    retry,
  } = useCyclesDetails({
    workspaceSlug,
    projectId,
    cycleId,
    isArchived,
  });

  if (isError) return <CatalogueReadFailure label="cycles" retry={retry} />;
  if (!cycleDetails) return <AnalyticsSidebarLoader className="px-5" />;

  return (
    <div className="relative pb-2">
      <div className="flex w-full flex-col gap-5">
        <CycleSidebarHeader
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          cycleDetails={cycleDetails}
          isArchived={isArchived}
          handleClose={handleClose}
        />
        <CycleSidebarDetails workspaceSlug={workspaceSlug} projectId={projectId} cycleDetails={cycleDetails} />
      </div>

      {workspaceSlug && projectId && cycleDetails?.id && (
        <CycleAnalyticsProgress
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          cycleId={cycleDetails?.id}
          isArchived={isArchived}
        />
      )}
    </div>
  );
};
