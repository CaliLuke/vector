/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { download, generateCsv, mkConfig } from "export-to-csv";
import type { RowData } from "@tanstack/react-table";
import type { AnalyticsColumnDef, AnalyticsRow } from "./insight-table/table-features";

const csvConfig = (workspaceSlug: string) =>
  mkConfig({
    fieldSeparator: ",",
    filename: `${workspaceSlug}-analytics`,
    decimalSeparator: ".",
    useKeysAsHeaders: true,
  });

export const exportCSV = <T extends RowData>(
  rows: AnalyticsRow<T>[],
  columns: AnalyticsColumnDef<T>[],
  workspaceSlug: string
) => {
  const rowData = rows.map((row) => {
    const exportColumns = columns.map((col) => col.meta?.export);
    const cells = exportColumns.reduce((acc: Record<string, string | number>, col) => {
      if (col) {
        const cell = col?.value(row) ?? "-";
        acc[col.label ?? col.key] = cell;
      }
      return acc;
    }, {});
    return cells;
  });
  const csv = generateCsv(csvConfig(workspaceSlug))(rowData);
  download(csvConfig(workspaceSlug))(csv);
};
