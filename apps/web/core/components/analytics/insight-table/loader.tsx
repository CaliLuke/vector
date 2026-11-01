/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import * as React from "react";
import range from "lodash-es/range.js";
import { useTable } from "@tanstack/react-table";
import { analyticsTableFeatures } from "./table-features";
import type { RowData } from "@tanstack/react-table";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@plane/propel/table";
import { Loader } from "@plane/ui";
import type { AnalyticsColumnDef } from "./table-features";

interface TableSkeletonProps<TData extends RowData> {
  columns: AnalyticsColumnDef<TData>[];
  rows: number;
}

export function TableLoader<TData extends RowData>({ columns, rows }: TableSkeletonProps<TData>) {
  const table = useTable({ features: analyticsTableFeatures, columns, data: [] });
  const leafColumns = table.getAllLeafColumns();
  return (
    <Table>
      <TableHeader>
        {table.getHeaderGroups().map((group) => (
          <TableRow key={group.id}>
            {group.headers.map((header) => (
              <TableHead key={header.id} colSpan={header.colSpan}>
                {!header.isPlaceholder && typeof header.column.columnDef.header === "string"
                  ? header.column.columnDef.header
                  : ""}
              </TableHead>
            ))}
          </TableRow>
        ))}
      </TableHeader>
      <TableBody>
        {range(rows).map((rowSlot) => (
          <TableRow key={rowSlot}>
            {leafColumns.map((column) => (
              <TableCell key={column.id}>
                <Loader.Item height="20px" width="100%" />
              </TableCell>
            ))}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
