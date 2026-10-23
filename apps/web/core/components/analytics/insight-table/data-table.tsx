/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import * as React from "react";
import type {
  ColumnFiltersState,
  ColumnVisibilityState,
  RowData,
  RowSelectionState,
  SortingState,
} from "@tanstack/react-table";
import { flexRender, useTable } from "@tanstack/react-table";

import { useTranslation } from "@plane/i18n";
import { EmptyStateCompact } from "@plane/propel/empty-state";
import { SearchIcon, CloseIcon } from "@plane/propel/icons";
// plane package imports
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@plane/propel/table";
import { cn } from "@plane/utils";
import { analyticsTableFeatures } from "./table-features";
import type { AnalyticsColumnDef, AnalyticsTable } from "./table-features";
// plane web components

interface DataTableProps<TData extends RowData> {
  columns: AnalyticsColumnDef<TData>[];
  data: TData[];
  searchPlaceholder: string;
  actions?: (table: AnalyticsTable<TData>) => React.ReactNode;
}

export function DataTable<TData extends RowData>({ columns, data, searchPlaceholder, actions }: DataTableProps<TData>) {
  const [rowSelection, _setRowSelection] = React.useState<RowSelectionState>({});
  const [columnVisibility, setColumnVisibility] = React.useState<ColumnVisibilityState>({});
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>([]);
  const [sorting, _setSorting] = React.useState<SortingState>([]);
  const { t } = useTranslation();
  const inputRef = React.useRef<HTMLInputElement>(null);
  const [isSearchOpen, setIsSearchOpen] = React.useState(false);

  const table = useTable({
    features: analyticsTableFeatures,
    data,
    columns,
    state: {
      sorting,
      columnVisibility,
      rowSelection,
      columnFilters,
    },
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
  });

  const searchColumn = table.getHeaderGroups()[0]?.headers[0]?.column;
  const searchFilter = searchColumn?.getFilterValue();

  return (
    <div className="space-y-4">
      <div className="flex w-full items-center justify-between">
        <div className="relative flex max-w-[300px] items-center gap-4">
          {searchColumn && (
            <div className="flex items-center gap-2 text-13 whitespace-nowrap text-placeholder">
              {searchPlaceholder}
            </div>
          )}
          {!isSearchOpen && (
            <button
              type="button"
              className="-mr-5 grid place-items-center rounded-sm p-2 text-placeholder hover:bg-layer-1"
              onClick={() => {
                setIsSearchOpen(true);
                inputRef.current?.focus();
              }}
            >
              <SearchIcon className="h-3.5 w-3.5" />
            </button>
          )}
          <div
            className={cn(
              "mr-auto flex w-0 items-center justify-start gap-1 overflow-hidden rounded-md border border-transparent bg-surface-1 text-placeholder opacity-0 transition-[width] ease-linear",
              {
                "w-64 border-subtle px-2.5 py-1.5 opacity-100": isSearchOpen,
              }
            )}
          >
            <SearchIcon className="h-3.5 w-3.5" />
            <input
              ref={inputRef}
              className="w-full max-w-[234px] border-none bg-transparent text-13 text-primary placeholder:text-placeholder focus:outline-none"
              placeholder="Search"
              value={typeof searchFilter === "string" ? searchFilter : ""}
              onChange={(e) => {
                searchColumn?.setFilterValue(e.target.value);
              }}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  setIsSearchOpen(true);
                }
              }}
            />
            {isSearchOpen && (
              <button
                type="button"
                className="grid place-items-center"
                onClick={() => {
                  searchColumn?.setFilterValue("");
                  setIsSearchOpen(false);
                }}
              >
                <CloseIcon className="h-3 w-3" />
              </button>
            )}
          </div>
        </div>
        {actions && <div>{actions(table)}</div>}
      </div>

      <div className="rounded-md">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <TableHead key={header.id} colSpan={header.colSpan} className="whitespace-nowrap">
                    {header.isPlaceholder
                      ? null
                      : (flexRender(header.column.columnDef.header, header.getContext()) as any)}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows?.length > 0 ? (
              table.getRowModel().rows.map((row) => (
                <TableRow key={row.id} data-state={row.getIsSelected() && "selected"}>
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      {flexRender(cell.column.columnDef.cell, cell.getContext()) as any}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={columns.length} className="p-0">
                  <EmptyStateCompact
                    assetKey="unknown"
                    assetClassName="size-20"
                    rootClassName="border border-subtle px-5 py-10 md:py-20 md:px-20"
                    title={t("workspace_empty_state.analytics_work_items.title")}
                  />
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
