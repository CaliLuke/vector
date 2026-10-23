/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import * as React from "react";
import type { RowData } from "@tanstack/react-table";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@plane/propel/table";
import { Loader } from "@plane/ui";
import type { AnalyticsColumnDef } from "./table-features";

interface TableSkeletonProps<TData extends RowData> {
  columns: AnalyticsColumnDef<TData>[];
  rows: number;
}

export function TableLoader<TData extends RowData>({ columns, rows }: TableSkeletonProps<TData>) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          {columns.map((column, index) => (
            <TableHead key={column.header?.toString() ?? index}>
              {typeof column.header === "string" ? column.header : ""}
            </TableHead>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        {Array.from({ length: rows }).map((_, rowIndex) => (
          <TableRow key={rowIndex}>
            {columns.map((_, colIndex) => (
              <TableCell key={colIndex}>
                <Loader.Item height="20px" width="100%" />
              </TableCell>
            ))}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
