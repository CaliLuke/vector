/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery } from "@tanstack/react-query";
// plane package imports
import { useTranslation } from "@plane/i18n";
import { Loader } from "@plane/ui";
// plane web hooks
import { useAnalyticsState } from "@/providers/analytics-state-provider";
import { projectAnalyticsCountsOptions } from "@/queries/analytics";
import { AnalyticsQueryError } from "../query-error";
// plane web components
import AnalyticsSectionWrapper from "../analytics-section-wrapper";
import ActiveProjectItem from "./active-project-item";

const ActiveProjects = function ActiveProjects() {
  const { t } = useTranslation();
  const { scope, selectedDurationLabel } = useAnalyticsState();
  const projectAnalyticsCountQuery = useQuery(
    projectAnalyticsCountsOptions(scope, { fields: "total_work_items,total_completed_work_items" })
  );
  const { data: projectAnalyticsCount, isLoading: isProjectAnalyticsCountLoading } = projectAnalyticsCountQuery;
  const canRead = Boolean(scope.workspaceRole && scope.membershipRevision);
  return (
    <AnalyticsSectionWrapper
      title={`${t("workspace_analytics.active_projects")}`}
      subtitle={selectedDurationLabel}
      className="md:col-span-2"
    >
      <div className="flex h-[350px] flex-col gap-4 overflow-auto">
        {!canRead ? (
          <output className="text-sm text-secondary">You do not have access to these analytics.</output>
        ) : projectAnalyticsCountQuery.isError ? (
          <AnalyticsQueryError onRetry={() => void projectAnalyticsCountQuery.refetch()} />
        ) : null}
        {canRead &&
          isProjectAnalyticsCountLoading &&
          Array.from({ length: 5 }).map((_, index) => <Loader.Item key={index} height="40px" width="100%" />)}
        {canRead &&
          !isProjectAnalyticsCountLoading &&
          !projectAnalyticsCountQuery.isError &&
          projectAnalyticsCount?.map((project) => <ActiveProjectItem key={project.id} project={project} />)}
      </div>
    </AnalyticsSectionWrapper>
  );
};

export default ActiveProjects;
