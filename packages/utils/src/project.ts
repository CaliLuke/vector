/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import sortBy from "lodash-es/sortBy.js";
// plane imports
import type { TProject, TProjectDisplayFilters, TProjectFilters, TProjectOrderByOptions } from "@plane/types";
// local imports
import { matchesFilterValue, matchesAnyFilterValue, matchesDateFilters } from "./filter";

/**
 * Updates the sort order of the project.
 * @param sortIndex
 * @param destinationIndex
 * @param projectId
 * @returns number | undefined
 */
export const orderJoinedProjects = (
  sourceIndex: number,
  destinationIndex: number,
  currentProjectId: string,
  joinedProjects: TProject[]
): number | undefined => {
  if (!currentProjectId || sourceIndex < 0 || destinationIndex < 0 || joinedProjects.length <= 0) return undefined;

  let updatedSortOrder: number | undefined = undefined;
  const sortOrderDefaultValue = 10000;

  if (destinationIndex === 0) {
    // updating project at the top of the project
    const currentSortOrder = joinedProjects[destinationIndex].sort_order || 0;
    updatedSortOrder = currentSortOrder - sortOrderDefaultValue;
  } else if (destinationIndex === joinedProjects.length) {
    // updating project at the bottom of the project
    const currentSortOrder = joinedProjects[destinationIndex - 1].sort_order || 0;
    updatedSortOrder = currentSortOrder + sortOrderDefaultValue;
  } else {
    // updating project in the middle of the project
    const destinationTopProjectSortOrder = joinedProjects[destinationIndex - 1].sort_order || 0;
    const destinationBottomProjectSortOrder = joinedProjects[destinationIndex].sort_order || 0;
    const updatedValue = (destinationTopProjectSortOrder + destinationBottomProjectSortOrder) / 2;
    updatedSortOrder = updatedValue;
  }

  return updatedSortOrder;
};

export const projectIdentifierSanitizer = (identifier: string): string =>
  identifier.replace(/[^ÇŞĞIİÖÜA-Za-z0-9]/g, "");

/**
 * @description filters projects based on the filter
 * @param {TProject} project
 * @param {TProjectFilters} filters
 * @param {TProjectDisplayFilters} displayFilters
 * @returns {boolean}
 */
export const shouldFilterProject = (
  project: TProject,
  displayFilters: TProjectDisplayFilters,
  filters: TProjectFilters
): boolean => {
  const lead =
    typeof project.project_lead === "object" && project.project_lead !== null
      ? project.project_lead.id
      : project.project_lead;
  return (
    matchesFilterValue(String(project.network), filters.access) &&
    matchesFilterValue(String(lead), filters.lead) &&
    matchesAnyFilterValue(project.members, filters.members) &&
    matchesDateFilters(project.created_at, filters.created_at) &&
    (!displayFilters.my_projects || Boolean(project.member_role)) &&
    Boolean(project.archived_at) === Boolean(displayFilters.archived_projects)
  );
};

/**
 * @description orders projects based on the orderByKey
 * @param {TProject[]} projects
 * @param {TProjectOrderByOptions | undefined} orderByKey
 * @returns {TProject[]}
 */
export const orderProjects = (projects: TProject[], orderByKey: TProjectOrderByOptions | undefined): TProject[] => {
  let orderedProjects: TProject[] = [];
  if (projects.length === 0) return orderedProjects;

  if (orderByKey === "sort_order") orderedProjects = sortBy(projects, [(p) => p.sort_order]);
  if (orderByKey === "name") orderedProjects = sortBy(projects, [(p) => p.name.toLowerCase()]);
  if (orderByKey === "-name") orderedProjects = sortBy(projects, [(p) => p.name.toLowerCase()]).toReversed();
  if (orderByKey === "created_at") orderedProjects = sortBy(projects, [(p) => p.created_at]);
  if (orderByKey === "-created_at") orderedProjects = sortBy(projects, [(p) => !p.created_at]);
  if (orderByKey === "members_length") orderedProjects = sortBy(projects, [(p) => p.members?.length]);
  if (orderByKey === "-members_length") orderedProjects = sortBy(projects, [(p) => p.members?.length]).toReversed();

  return orderedProjects;
};
