/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane web components
import { useParams } from "@tanstack/react-router";

// hooks
import { useAnalyticsState } from "@/providers/analytics-state-provider";
import { useJoinedProjectIds } from "@/hooks/use-joined-project-ids";
// components
import { ProjectSelect } from "./select/project";

const AnalyticsFilterActions = function AnalyticsFilterActions() {
  const { selectedProjects, updateSelectedProjects } = useAnalyticsState();
  const { workspaceSlug } = useParams({ strict: false });
  const joinedProjectIds = useJoinedProjectIds(workspaceSlug?.toString());
  return (
    <div className="flex items-center justify-end gap-2">
      <ProjectSelect
        value={selectedProjects}
        onChange={(val) => {
          updateSelectedProjects(val ?? []);
        }}
        projectIds={joinedProjectIds}
      />
    </div>
  );
};

export default AnalyticsFilterActions;
