/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

type HeaderProps = {
  column: { columnDef: { meta?: { align?: "left" | "right"; export: { key: string; label?: string } } } };
};

export function AnalyticsColumnHeader({ column }: HeaderProps) {
  const meta = column.columnDef.meta;
  return (
    <div className={meta?.align === "right" ? "text-right" : "text-left"}>{meta?.export.label ?? meta?.export.key}</div>
  );
}

export function AnalyticsValueCell({ getValue }: { getValue: () => unknown }) {
  const value = getValue();
  return <div className="text-right">{typeof value === "number" || typeof value === "string" ? value : null}</div>;
}
