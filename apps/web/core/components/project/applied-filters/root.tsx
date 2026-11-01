/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ClearAppliedFilters, RemoveAppliedFilter } from "@/components/common/applied-filters/tag";
import { orderedFilterKeys } from "@/components/common/applied-filters/keys";
import { useParams } from "@tanstack/react-router";
import { PROJECT_CREATED_AT_FILTER_OPTIONS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
// plane imports
import { Tooltip } from "@plane/propel/tooltip";
import type { TProjectAppliedDisplayFilterKeys, TProjectFilters } from "@plane/types";
import { EHeaderVariant, Header, Tag } from "@plane/ui";
import { replaceUnderscoreIfSnakeCase } from "@plane/utils/string";
// local imports
import { AppliedAccessFilters } from "./access";
import { AppliedDateFilters } from "@/components/common/applied-filters/date";
import { AppliedMembersFilters } from "@/components/common/applied-filters/members";
import { AppliedProjectDisplayFilters } from "./project-display-filters";

const FILTER_KEYS = ["access", "lead", "members", "created_at"] as const satisfies readonly (keyof TProjectFilters)[];

type Props = {
  appliedFilters: TProjectFilters;
  appliedDisplayFilters: TProjectAppliedDisplayFilterKeys[];
  handleClearAllFilters: () => void;
  handleRemoveFilter: (key: keyof TProjectFilters, value: string | null) => void;
  handleRemoveDisplayFilter: (key: TProjectAppliedDisplayFilterKeys) => void;
  alwaysAllowEditing?: boolean;
  filteredProjects: number;
  totalProjects: number;
};

const MEMBERS_FILTERS = new Set(["lead", "members"]);
const DATE_FILTERS = new Set(["created_at"]);

export function ProjectAppliedFiltersList(props: Props) {
  const { workspaceSlug } = useParams({ strict: false });
  const { t } = useTranslation();
  const {
    appliedFilters,
    appliedDisplayFilters,
    handleClearAllFilters,
    handleRemoveFilter,
    handleRemoveDisplayFilter,
    alwaysAllowEditing,
    filteredProjects,
    totalProjects,
  } = props;

  if (!appliedFilters && !appliedDisplayFilters) return null;
  if (Object.keys(appliedFilters).length === 0 && appliedDisplayFilters.length === 0) return null;

  const isEditingAllowed = alwaysAllowEditing;

  return (
    <Header variant={EHeaderVariant.TERNARY}>
      <Header.LeftItem>
        {/* Applied filters */}
        {orderedFilterKeys(appliedFilters ?? {}, FILTER_KEYS).map((filterKey) => {
          const value = appliedFilters[filterKey];

          if (!value) return undefined;
          if (Array.isArray(value) && value.length === 0) return undefined;

          return (
            <Tag key={filterKey}>
              <span className="text-11 text-tertiary">{replaceUnderscoreIfSnakeCase(filterKey)}</span>
              {filterKey === "access" && (
                <AppliedAccessFilters
                  editable={isEditingAllowed}
                  handleRemove={(val) => handleRemoveFilter("access", val)}
                  values={value}
                />
              )}
              {DATE_FILTERS.has(filterKey) && (
                <AppliedDateFilters
                  options={PROJECT_CREATED_AT_FILTER_OPTIONS}
                  editable={isEditingAllowed}
                  handleRemove={(val) => handleRemoveFilter(filterKey, val)}
                  values={value}
                />
              )}
              {MEMBERS_FILTERS.has(filterKey) && (
                <AppliedMembersFilters
                  workspaceSlug={workspaceSlug}
                  editable={isEditingAllowed}
                  handleRemove={(val) => handleRemoveFilter(filterKey, val)}
                  values={value}
                />
              )}
              {isEditingAllowed && (
                <RemoveAppliedFilter filterKey={filterKey} onRemove={() => handleRemoveFilter(filterKey, null)} />
              )}
            </Tag>
          );
        })}
        {/* Applied display filters */}
        {appliedDisplayFilters.length > 0 && (
          <Tag key="project_display_filters">
            <span className="text-11 text-tertiary">{t("common.projects")}</span>
            <AppliedProjectDisplayFilters
              editable={isEditingAllowed}
              values={appliedDisplayFilters}
              handleRemove={(key) => handleRemoveDisplayFilter(key)}
            />
          </Tag>
        )}
        {isEditingAllowed && <ClearAppliedFilters onClear={handleClearAllFilters} />}
      </Header.LeftItem>
      <Header.RightItem>
        <Tooltip
          tooltipContent={
            <p>
              <span className="font-semibold">{filteredProjects}</span> of{" "}
              <span className="font-semibold">{totalProjects}</span> projects match the applied filters.
            </p>
          }
        >
          <span className="rounded-full bg-layer-1 px-2.5 py-1 text-13 font-medium">
            {filteredProjects}/{totalProjects}
          </span>
        </Tooltip>
      </Header.RightItem>
    </Header>
  );
}
