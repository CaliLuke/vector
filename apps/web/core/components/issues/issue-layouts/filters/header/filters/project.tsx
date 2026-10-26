/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCurrentWorkspace } from "@/hooks/use-workspaces";
import React, { useMemo } from "react";
import sortBy from "lodash-es/sortBy.js";

// ui
import { Logo } from "@plane/propel/emoji-icon-picker";
// components
import { FilterOption } from "@/components/issues/issue-layouts/filters/header/helpers/filter-option";
import { FilterOptionSection } from "@/components/issues/issue-layouts/filters/header/helpers/filter-option-section";
// hooks
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useJoinedProjectIds } from "@/hooks/use-joined-project-ids";

type Props = {
  appliedFilters: string[] | null;
  handleUpdate: (val: string) => void;
  searchQuery: string;
};

export const FilterProjects = function FilterProjects(props: Props) {
  const { appliedFilters, handleUpdate, searchQuery } = props;
  // states
  // store
  const { data: catalogueWorkspace } = useCurrentWorkspace();
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  const joinedProjectIds = useJoinedProjectIds(catalogueWorkspace?.slug);
  // derived values
  const projects =
    joinedProjectIds?.flatMap((projectId) => {
      const project = projectRows?.find((row) => row.id === projectId);
      return project ? [project] : [];
    }) ?? null;
  const appliedFiltersCount = appliedFilters?.length ?? 0;

  const sortedOptions = useMemo(() => {
    if (!projects) return null;

    const filteredOptions = projects.filter((project) =>
      project.name.toLowerCase().includes(searchQuery.toLowerCase())
    );
    return sortBy(filteredOptions, [
      (project) => !(appliedFilters ?? []).includes(project.id),
      (project) => project.name.toLowerCase(),
    ]);
  }, [appliedFilters, projects, searchQuery]);

  return (
    <FilterOptionSection
      title={`Project${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
      optionCount={sortedOptions?.length ?? 0}
      state={sortedOptions === null ? { kind: "loading" } : undefined}
      renderOptions={(itemsToRender) =>
        (sortedOptions ?? []).slice(0, itemsToRender).map((project) => (
          <FilterOption
            key={`project-${project.id}`}
            isChecked={appliedFilters?.includes(project.id) ?? false}
            onClick={() => handleUpdate(project.id)}
            icon={
              <span className="grid h-4 w-4 flex-shrink-0 place-items-center">
                <Logo logo={project.logo_props} size={12} />
              </span>
            }
            title={project.name}
          />
        ))
      }
    />
  );
};
