/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import type { IIssueDisplayFilterOptions, IssuePaginationOptions } from "@plane/types";
import sortBy from "lodash-es/sortBy.js";
import { hashKey } from "@tanstack/react-query";
import { useParams } from "@tanstack/react-router";
import { isWorkspaceLevel } from "@/components/issues/issue-layouts/utils";
import { useIssueLayoutLabels } from "@/hooks/use-issue-layout-labels";
import { useIssueEstimateProjection } from "@/hooks/use-issue-estimate-projection";
import { useIssueParentCatalogues } from "@/hooks/use-issue-parent-catalogues";
import { useIssueStateOrder } from "@/hooks/use-project-states";
import { useWorkspaceProjects } from "@/hooks/use-projects";
import { useIssueListRead } from "@/hooks/use-issue-list-read";
import { useIssueListQuery } from "@/hooks/use-issue-list-query";
import type { NativeIssueStoreType } from "@/hooks/use-issue-filter-preferences";

function labelReadRequirements(displayFilters: IIssueDisplayFilterOptions | undefined) {
  const grouped = displayFilters?.group_by === "labels" || displayFilters?.sub_group_by === "labels";
  const names = grouped || displayFilters?.order_by === "labels__name" || displayFilters?.order_by === "-labels__name";
  return { grouped, names };
}

function issueOrderingNeedsEstimate(displayFilters: IIssueDisplayFilterOptions | undefined) {
  return displayFilters?.order_by === "estimate_point__key" || displayFilters?.order_by === "-estimate_point__key";
}

function issueOrderingNeedsProject(displayFilters: IIssueDisplayFilterOptions | undefined) {
  return displayFilters?.group_by === "project" || displayFilters?.sub_group_by === "project";
}

function hasProjectError(required: boolean, projects: ReturnType<typeof useWorkspaceProjects>) {
  return required && (projects.isError || projects.accessError || projects.accessDenied);
}

function isProjectReady(required: boolean, error: boolean, projects: ReturnType<typeof useWorkspaceProjects>) {
  return !required || (!error && projects.projects !== undefined);
}

function isStructuralReady(
  labelsRequired: ReturnType<typeof labelReadRequirements>,
  labelsReady: boolean,
  parents: ReturnType<typeof useIssueParentCatalogues>,
  projectRequired: boolean,
  projectReady: boolean
) {
  return (
    (!labelsRequired.grouped || labelsReady) &&
    (!parents.grouped || (!parents.isPending && !parents.isError)) &&
    (!projectRequired || projectReady)
  );
}

function hasPresentationError(input: {
  query: ReturnType<typeof useIssueListQuery>;
  labels: ReturnType<typeof useIssueLayoutLabels>;
  labelRequirements: ReturnType<typeof labelReadRequirements>;
  parents: ReturnType<typeof useIssueParentCatalogues>;
  projectError: boolean;
  estimateRequired: boolean;
  estimate: ReturnType<typeof useIssueEstimateProjection>;
  state: ReturnType<typeof useIssueStateOrder>;
}) {
  const { query, labels, labelRequirements, parents, projectError, estimateRequired, estimate, state } = input;
  return (
    query.isError ||
    labels.scopeIsError ||
    ((labelRequirements.names || labelRequirements.grouped) && (labels.isError || labels.isAccessDenied)) ||
    parents.isError ||
    projectError ||
    (estimateRequired && (estimate.isError || estimate.isAccessDenied)) ||
    state.isError
  );
}

function isPresentationPending(input: {
  isError: boolean;
  query: ReturnType<typeof useIssueListQuery>;
  structuralReady: boolean;
  displayReady: boolean;
}) {
  return !input.isError && (input.query.isPending || !input.structuralReady || !input.displayReady);
}

async function retryPresentation(input: {
  labels: ReturnType<typeof useIssueLayoutLabels>;
  labelRequirements: ReturnType<typeof labelReadRequirements>;
  parents: ReturnType<typeof useIssueParentCatalogues>;
  projectRequired: boolean;
  projects: ReturnType<typeof useWorkspaceProjects>;
  estimateRequired: boolean;
  estimate: ReturnType<typeof useIssueEstimateProjection>;
  state: ReturnType<typeof useIssueStateOrder>;
  query: ReturnType<typeof useIssueListQuery>;
}) {
  const { labels, labelRequirements, parents, projectRequired, projects, estimateRequired, estimate, state, query } =
    input;
  if (labels.scopeIsError || labelRequirements.names || labelRequirements.grouped) await labels.retry();
  if (parents.required) await parents.retry();
  if (projectRequired) {
    if (projects.accessError || projects.accessDenied) await projects.retryAccess();
    else await projects.refetch();
  }
  if (estimateRequired) await estimate.retry();
  await state.retry();
  if (query.isError) await query.retry();
}

function presentationReadiness(input: {
  query: ReturnType<typeof useIssueListQuery>;
  labels: ReturnType<typeof useIssueLayoutLabels>;
  labelRequirements: ReturnType<typeof labelReadRequirements>;
  parents: ReturnType<typeof useIssueParentCatalogues>;
  projectRequired: boolean;
  projects: ReturnType<typeof useWorkspaceProjects>;
  estimateRequired: boolean;
  estimate: ReturnType<typeof useIssueEstimateProjection>;
  state: ReturnType<typeof useIssueStateOrder>;
  structuralReady: boolean;
}) {
  const {
    query,
    labels,
    labelRequirements,
    parents,
    projectRequired,
    projects,
    estimateRequired,
    estimate,
    state,
    structuralReady,
  } = input;
  const projectError = hasProjectError(projectRequired, projects);
  const projectReady = isProjectReady(projectRequired, projectError, projects);
  const projectLabelReady = labels.scopeIsReady && !labels.isPending && !labels.isError && !labels.isAccessDenied;
  const displayReady =
    (!labelRequirements.names || projectLabelReady) &&
    !parents.isPending &&
    !parents.isError &&
    projectReady &&
    (!estimateRequired || (!estimate.isPending && !estimate.isError && !estimate.isAccessDenied)) &&
    !state.isPending &&
    !state.isError;
  const isError = hasPresentationError({
    query,
    labels,
    labelRequirements,
    parents,
    projectError,
    estimateRequired,
    estimate,
    state,
  });
  return {
    isError,
    isPending: isPresentationPending({ isError, query, structuralReady, displayReady }),
  };
}

export function useIssueListPresentation(
  storeType: NativeIssueStoreType,
  paginationOptions: IssuePaginationOptions | undefined,
  displayFilters: IIssueDisplayFilterOptions | undefined
) {
  const { workspaceSlug, projectId } = useParams({ strict: false });
  const workspaceLevel = isWorkspaceLevel(storeType);
  const labelRequirements = labelReadRequirements(displayFilters);
  const projectRequired = issueOrderingNeedsProject(displayFilters);
  const estimateRequired = issueOrderingNeedsEstimate(displayFilters);
  const labels = useIssueLayoutLabels(workspaceSlug, projectId, workspaceLevel);
  const parents = useIssueParentCatalogues(workspaceSlug, projectId, workspaceLevel, displayFilters);
  const projects = useWorkspaceProjects(projectRequired ? workspaceSlug : undefined);
  const estimate = useIssueEstimateProjection(workspaceSlug, projectId, workspaceLevel);
  const state = useIssueStateOrder(workspaceSlug, projectId, workspaceLevel, displayFilters?.order_by);

  const projectError = hasProjectError(projectRequired, projects);
  const projectReady = isProjectReady(projectRequired, projectError, projects);
  const projectLabelReady = labels.scopeIsReady && !labels.isPending && !labels.isError && !labels.isAccessDenied;
  const structuralReady = isStructuralReady(
    labelRequirements,
    projectLabelReady,
    parents,
    projectRequired,
    projectReady
  );
  const catalogueRevision = hashKey([
    labelRequirements.grouped ? labels.idsRevision : undefined,
    parents.grouped ? parents.structuralRevision : undefined,
    projectRequired ? sortBy(projects.projects?.map((project) => project.id) ?? [], (id) => id) : undefined,
  ]);
  const read = useIssueListRead(
    storeType,
    paginationOptions ?? { canGroup: false, perPageCount: 100 },
    structuralReady && paginationOptions !== undefined,
    catalogueRevision
  );
  const query = useIssueListQuery(read);

  const retry = () =>
    retryPresentation({
      labels,
      labelRequirements,
      parents,
      projectRequired,
      projects,
      estimateRequired,
      estimate,
      state,
      query,
    });

  const issueCount = Object.keys(query.issuesById).length;
  const readiness = presentationReadiness({
    query,
    labels,
    labelRequirements,
    parents,
    projectRequired,
    projects,
    estimateRequired,
    estimate,
    state,
    structuralReady,
  });

  return {
    ...query,
    issueCount,
    labelsById: labels.labelsById,
    estimateProjection: estimate.projection,
    stateProjection: state.projection,
    parentProjection: parents.parents,
    isPending: readiness.isPending,
    isError: readiness.isError,
    retry,
  };
}
