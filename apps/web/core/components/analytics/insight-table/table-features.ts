/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import {
  columnFacetingFeature,
  columnFilteringFeature,
  columnVisibilityFeature,
  createFacetedRowModel,
  createFacetedUniqueValues,
  createFilteredRowModel,
  createSortedRowModel,
  filterFn_includesString,
  rowSelectionFeature,
  rowSortingFeature,
  sortFn_alphanumeric,
  sortFn_text,
  tableFeatures,
} from "@tanstack/react-table";
import type { CellData, ColumnDef, Row, RowData, Table, TableFeatures } from "@tanstack/react-table";

export const analyticsTableFeatures = tableFeatures({
  columnFacetingFeature,
  columnFilteringFeature,
  columnVisibilityFeature,
  rowSelectionFeature,
  rowSortingFeature,
  filteredRowModel: createFilteredRowModel(),
  sortedRowModel: createSortedRowModel(),
  facetedRowModel: createFacetedRowModel(),
  facetedUniqueValues: createFacetedUniqueValues(),
  filterFns: { includesString: filterFn_includesString },
  sortFns: { alphanumeric: sortFn_alphanumeric, text: sortFn_text },
});

type AnalyticsTableFeatures = typeof analyticsTableFeatures;

export type AnalyticsColumnDef<TData extends RowData, TValue extends CellData = CellData> = ColumnDef<
  AnalyticsTableFeatures,
  TData,
  TValue
>;
export type AnalyticsRow<TData extends RowData> = Row<AnalyticsTableFeatures, TData>;
export type AnalyticsTable<TData extends RowData> = Table<AnalyticsTableFeatures, TData>;

declare module "@tanstack/react-table" {
  interface ColumnMeta<
    in out TFeatures extends TableFeatures,
    in out TData extends RowData,
    TValue extends CellData = CellData,
  > {
    export: {
      key: string;
      value: (row: Row<TFeatures, TData>) => string | number;
      label?: string;
    };
  }
}
