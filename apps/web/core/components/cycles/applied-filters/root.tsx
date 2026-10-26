/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// plane imports
import { EUserPermissions, DATE_AFTER_FILTER_OPTIONS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { CloseIcon } from "@plane/propel/icons";
import type { TCycleFilters } from "@plane/types";
import { Tag } from "@plane/ui";
import { replaceUnderscoreIfSnakeCase } from "@plane/utils/string";
// hooks
import { useProjectPermissions } from "@/hooks/use-permissions";
// local imports
import { AppliedDateFilters } from "@/components/common/applied-filters/date";
import { AppliedStatusFilters } from "./status";

type Props = {
  appliedFilters: TCycleFilters;
  handleClearAllFilters: () => void;
  handleRemoveFilter: (key: keyof TCycleFilters, value: string | null) => void;
  alwaysAllowEditing?: boolean;
};

const DATE_FILTERS = new Set<string>(["start_date", "end_date"]);

export const CycleAppliedFiltersList = function CycleAppliedFiltersList(props: Props) {
  const { appliedFilters, handleClearAllFilters, handleRemoveFilter, alwaysAllowEditing } = props;
  // store hooks
  const { workspaceSlug, projectId } = useParams({ strict: false });
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  const { t } = useTranslation();

  if (!appliedFilters) return null;

  if (Object.keys(appliedFilters).length === 0) return null;

  const isEditingAllowed = alwaysAllowEditing || canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  return (
    <div className="flex flex-wrap items-stretch gap-2 bg-surface-1">
      {Object.entries(appliedFilters).map(([key, value]) => {
        const filterKey = key as keyof TCycleFilters;

        if (!value) return;
        if (Array.isArray(value) && value.length === 0) return;

        return (
          <Tag key={filterKey}>
            <span className="text-11 text-tertiary">{replaceUnderscoreIfSnakeCase(filterKey)}</span>
            <div className="flex flex-wrap items-center gap-1">
              {filterKey === "status" && (
                <AppliedStatusFilters
                  editable={isEditingAllowed}
                  handleRemove={(val) => handleRemoveFilter("status", val)}
                  values={value}
                />
              )}
              {DATE_FILTERS.has(filterKey) && (
                <AppliedDateFilters
                  options={DATE_AFTER_FILTER_OPTIONS}
                  className="bg-layer-3"
                  editable={isEditingAllowed}
                  handleRemove={(val) => handleRemoveFilter(filterKey, val)}
                  values={value}
                />
              )}
              {isEditingAllowed && (
                <button
                  type="button"
                  className="grid place-items-center text-tertiary hover:text-secondary"
                  onClick={() => handleRemoveFilter(filterKey, null)}
                >
                  <CloseIcon height={12} width={12} strokeWidth={2} />
                </button>
              )}
            </div>
          </Tag>
        );
      })}
      {isEditingAllowed && (
        <button type="button" onClick={handleClearAllFilters}>
          <Tag>
            {t("common.clear_all")}
            <CloseIcon height={12} width={12} strokeWidth={2} />
          </Tag>
        </button>
      )}
    </div>
  );
};
