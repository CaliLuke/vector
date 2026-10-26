/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { useParams } from "@tanstack/react-router";
import type { TProjectDisplayFilters, TProjectFilters } from "@plane/types";
import { useTranslation } from "@plane/i18n";
import { ListFilter } from "lucide-react";
import { calculateTotalFilters } from "@plane/utils/filter";
import { FiltersDropdown } from "@/components/issues/issue-layouts/filters";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useProjectListFilters } from "@/hooks/use-project-list-filters";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { workspaceRosterMemberIds } from "@/helpers/member-rosters";
import { ProjectFiltersSelection } from "./root";

type Props = {
  menuButton?: ReactNode;
};

export function ProjectFiltersDropdown(props: Props) {
  const { menuButton } = props;
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const { t } = useTranslation();
  const { data: currentUser } = useCurrentUser();
  const { members } = useWorkspaceMemberRoster(workspaceSlug);
  const { displayFilters, filters, updateFilters, updateDisplayFilters } = useProjectListFilters();
  const memberIds = workspaceRosterMemberIds(members, currentUser?.id);
  const handleFiltersUpdate = (key: keyof TProjectFilters, value: string | string[]) => {
    if (!workspaceSlug) return;
    let values = [...(filters?.[key] ?? [])];
    if (Array.isArray(value)) {
      if (key === "created_at" && values.some((filter) => filter.includes("custom"))) values = [];
      for (const selected of value) {
        const index = values.indexOf(selected);
        if (index === -1) values.push(selected);
        else values.splice(index, 1);
      }
    } else {
      const index = values.indexOf(value);
      if (index !== -1) values.splice(index, 1);
      else if (key === "created_at") values = [value];
      else values.push(value);
    }
    updateFilters({ [key]: values });
  };
  const handleDisplayFiltersUpdate = (updated: Partial<TProjectDisplayFilters>) => {
    if (workspaceSlug) updateDisplayFilters(updated);
  };

  return (
    <FiltersDropdown
      icon={<ListFilter className="h-3 w-3" />}
      title={t("common.filters")}
      placement="bottom-end"
      menuButton={menuButton ?? null}
      isFiltersApplied={calculateTotalFilters(filters ?? {}) !== 0}
    >
      <ProjectFiltersSelection
        displayFilters={displayFilters ?? {}}
        filters={filters ?? {}}
        handleFiltersUpdate={handleFiltersUpdate}
        handleDisplayFiltersUpdate={handleDisplayFiltersUpdate}
        memberIds={memberIds}
        workspaceSlug={workspaceSlug}
      />
    </FiltersDropdown>
  );
}
