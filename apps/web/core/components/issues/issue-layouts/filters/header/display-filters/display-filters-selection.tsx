/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { availableExtraOptions } from "./extra-options";
import React from "react";
import { useTypeGroupingOptions } from "@/hooks/use-work-item-grouping-catalog";
import { CustomPropertyColumns } from "./custom-property-columns";
import isEmpty from "lodash-es/isEmpty.js";

import type {
  IIssueDisplayFilterOptions,
  IIssueDisplayProperties,
  ILayoutDisplayFiltersOptions,
  TIssueGroupByOptions,
} from "@plane/types";
// components
import {
  FilterDisplayProperties,
  FilterExtraOptions,
  FilterGroupBy,
  FilterOrderBy,
  FilterSubGroupBy,
} from "@/components/issues/issue-layouts/filters";

type Props = {
  displayFilters: IIssueDisplayFilterOptions | undefined;
  displayProperties: IIssueDisplayProperties;
  handleDisplayFiltersUpdate: (updatedDisplayFilter: Partial<IIssueDisplayFilterOptions>) => void;
  handleDisplayPropertiesUpdate: (updatedDisplayProperties: Partial<IIssueDisplayProperties>) => void;
  layoutDisplayFiltersOptions: ILayoutDisplayFiltersOptions | undefined;
  ignoreGroupedFilters?: Partial<TIssueGroupByOptions>[];
  cycleViewDisabled?: boolean;
  moduleViewDisabled?: boolean;
  isEpic?: boolean;
};

export const DisplayFiltersSelection = function DisplayFiltersSelection(props: Props) {
  const {
    displayFilters,
    displayProperties,
    handleDisplayFiltersUpdate,
    handleDisplayPropertiesUpdate,
    layoutDisplayFiltersOptions,
    ignoreGroupedFilters = [],
    cycleViewDisabled = false,
    moduleViewDisabled = false,
    isEpic = false,
  } = props;

  const layout = displayFilters?.layout;
  const groupingOptions = useTypeGroupingOptions(layout, isEpic);

  const isDisplayFilterEnabled = (displayFilter: keyof IIssueDisplayFilterOptions) =>
    Object.keys(layoutDisplayFiltersOptions?.display_filters ?? {}).includes(displayFilter);

  const computedIgnoreGroupedFilters: Partial<TIssueGroupByOptions>[] = [...ignoreGroupedFilters];
  if (cycleViewDisabled) {
    computedIgnoreGroupedFilters.push("cycle");
  }
  if (moduleViewDisabled) {
    computedIgnoreGroupedFilters.push("module");
  }

  return (
    <div className="vertical-scrollbar relative scrollbar-sm h-full w-full divide-y divide-subtle-1 overflow-hidden overflow-y-auto px-2.5">
      {/* display properties */}
      {layoutDisplayFiltersOptions?.display_properties && layoutDisplayFiltersOptions.display_properties.length > 0 && (
        <div className="py-2">
          <FilterDisplayProperties
            displayProperties={displayProperties}
            displayPropertiesToRender={layoutDisplayFiltersOptions.display_properties}
            handleUpdate={handleDisplayPropertiesUpdate}
            cycleViewDisabled={cycleViewDisabled}
            moduleViewDisabled={moduleViewDisabled}
            isEpic={isEpic}
          />
        </div>
      )}

      <CustomPropertyColumns
        displayProperties={displayProperties}
        displayFilters={displayFilters}
        isEpic={isEpic}
        onChange={handleDisplayPropertiesUpdate}
      />
      {/* group by */}
      {isDisplayFilterEnabled("group_by") && (
        <div className="py-2">
          <FilterGroupBy
            isEpic={isEpic}
            displayFilters={displayFilters}
            groupByOptions={groupingOptions(layoutDisplayFiltersOptions?.display_filters.group_by ?? [])}
            handleUpdate={(val) =>
              handleDisplayFiltersUpdate({
                group_by: val,
              })
            }
            ignoreGroupedFilters={computedIgnoreGroupedFilters}
          />
        </div>
      )}

      {/* sub-group by */}
      {isDisplayFilterEnabled("sub_group_by") &&
        displayFilters &&
        displayFilters.group_by !== null &&
        layout === "kanban" && (
          <div className="py-2">
            <FilterSubGroupBy
              isEpic={isEpic}
              displayFilters={displayFilters}
              handleUpdate={(val) =>
                handleDisplayFiltersUpdate({
                  sub_group_by: val,
                })
              }
              subGroupByOptions={groupingOptions(layoutDisplayFiltersOptions?.display_filters.sub_group_by ?? [])}
              ignoreGroupedFilters={computedIgnoreGroupedFilters}
            />
          </div>
        )}

      {/* order by */}
      {isDisplayFilterEnabled("order_by") && !isEmpty(layoutDisplayFiltersOptions?.display_filters?.order_by) && (
        <div className="py-2">
          <FilterOrderBy
            selectedOrderBy={displayFilters?.order_by}
            handleUpdate={(val) =>
              handleDisplayFiltersUpdate({
                order_by: val,
              })
            }
            orderByOptions={layoutDisplayFiltersOptions?.display_filters.order_by ?? []}
          />
        </div>
      )}

      {/* Options */}
      {layoutDisplayFiltersOptions?.extra_options.access && (
        <div className="py-2">
          <FilterExtraOptions
            selectedExtraOptions={{
              show_empty_groups: displayFilters?.show_empty_groups ?? true,
              sub_issue: displayFilters?.sub_issue ?? true,
            }}
            handleUpdate={(key, val) =>
              handleDisplayFiltersUpdate({
                [key]: val,
              })
            }
            enabledExtraOptions={availableExtraOptions(
              layoutDisplayFiltersOptions.extra_options.values,
              displayFilters
            )}
          />
        </div>
      )}
    </div>
  );
};
