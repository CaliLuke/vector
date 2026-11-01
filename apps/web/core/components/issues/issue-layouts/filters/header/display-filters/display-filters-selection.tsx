/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { EIssueLayoutTypes } from "@plane/types";
import { availableExtraOptions } from "./extra-options";
import React from "react";
import { useTypeGroupingOptions } from "@/hooks/use-work-item-grouping-catalog";
import { CustomPropertyColumns } from "./custom-property-columns";

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

export function DisplayFiltersSelection(props: Props) {
  const {
    displayFilters,
    displayProperties,
    handleDisplayPropertiesUpdate,
    layoutDisplayFiltersOptions,
    cycleViewDisabled = false,
    moduleViewDisabled = false,
    isEpic = false,
  } = props;
  const properties = layoutDisplayFiltersOptions?.display_properties ?? [];
  return (
    <div className="vertical-scrollbar relative scrollbar-sm h-full w-full divide-y divide-subtle-1 overflow-hidden overflow-y-auto px-2.5">
      {properties.length > 0 && (
        <div className="py-2">
          <FilterDisplayProperties
            displayProperties={displayProperties}
            displayPropertiesToRender={properties}
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
      <DisplayGroupingFilters {...props} />
      <DisplayOrderFilter {...props} />
      <DisplayExtraOptions {...props} />
    </div>
  );
}

const NO_IGNORED_GROUPS: Partial<TIssueGroupByOptions>[] = [];

function DisplayGroupingFilters({
  displayFilters,
  handleDisplayFiltersUpdate,
  layoutDisplayFiltersOptions,
  ignoreGroupedFilters = NO_IGNORED_GROUPS,
  cycleViewDisabled = false,
  moduleViewDisabled = false,
  isEpic = false,
}: Props) {
  const layout = displayFilters?.layout;
  const groupingOptions = useTypeGroupingOptions(layout, isEpic);
  const options = layoutDisplayFiltersOptions?.display_filters;
  const ignored = [...ignoreGroupedFilters];
  if (cycleViewDisabled) ignored.push("cycle");
  if (moduleViewDisabled) ignored.push("module");
  return (
    <>
      {options && Object.hasOwn(options, "group_by") && (
        <div className="py-2">
          <FilterGroupBy
            isEpic={isEpic}
            displayFilters={displayFilters}
            groupByOptions={groupingOptions(options.group_by ?? [])}
            handleUpdate={(group_by) => handleDisplayFiltersUpdate({ group_by })}
            ignoreGroupedFilters={ignored}
          />
        </div>
      )}
      <DisplaySubGroupFilter
        displayFilters={displayFilters}
        options={options}
        groupingOptions={groupingOptions}
        ignored={ignored}
        onChange={handleDisplayFiltersUpdate}
        isEpic={isEpic}
      />
    </>
  );
}

function DisplaySubGroupFilter({
  displayFilters,
  options,
  groupingOptions,
  ignored,
  onChange,
  isEpic,
}: {
  displayFilters: Props["displayFilters"];
  options: ILayoutDisplayFiltersOptions["display_filters"] | undefined;
  groupingOptions: ReturnType<typeof useTypeGroupingOptions>;
  ignored: Partial<TIssueGroupByOptions>[];
  onChange: Props["handleDisplayFiltersUpdate"];
  isEpic: boolean;
}) {
  if (
    !options ||
    !Object.hasOwn(options, "sub_group_by") ||
    !displayFilters ||
    displayFilters.group_by === null ||
    displayFilters.layout !== EIssueLayoutTypes.KANBAN
  )
    return null;
  return (
    <div className="py-2">
      <FilterSubGroupBy
        isEpic={isEpic}
        displayFilters={displayFilters}
        handleUpdate={(sub_group_by) => onChange({ sub_group_by })}
        subGroupByOptions={groupingOptions(options.sub_group_by ?? [])}
        ignoreGroupedFilters={ignored}
      />
    </div>
  );
}

function DisplayOrderFilter({ displayFilters, layoutDisplayFiltersOptions, handleDisplayFiltersUpdate }: Props) {
  const options = layoutDisplayFiltersOptions?.display_filters.order_by;
  if (!options?.length) return null;
  return (
    <div className="py-2">
      <FilterOrderBy
        selectedOrderBy={displayFilters?.order_by}
        handleUpdate={(order_by) => handleDisplayFiltersUpdate({ order_by })}
        orderByOptions={options}
      />
    </div>
  );
}

function DisplayExtraOptions({ displayFilters, layoutDisplayFiltersOptions, handleDisplayFiltersUpdate }: Props) {
  const options = layoutDisplayFiltersOptions?.extra_options;
  if (!options?.access) return null;
  return (
    <div className="py-2">
      <FilterExtraOptions
        selectedExtraOptions={{
          show_empty_groups: displayFilters?.show_empty_groups ?? true,
          sub_issue: displayFilters?.sub_issue ?? true,
        }}
        handleUpdate={(key, value) => handleDisplayFiltersUpdate({ [key]: value })}
        enabledExtraOptions={availableExtraOptions(options.values, displayFilters)}
      />
    </div>
  );
}
