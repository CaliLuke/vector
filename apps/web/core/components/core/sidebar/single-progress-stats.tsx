/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";

type TSingleProgressStatsProps = {
  title: React.ReactNode;
  completed: number;
  total: number;
  onClick?: () => void;
  selected?: boolean;
};

export function SingleProgressStats({ title, completed, total, onClick, selected = false }: TSingleProgressStatsProps) {
  const className = `flex w-full items-center justify-between gap-4 rounded-xs p-1 text-left text-11 ${
    onClick ? "cursor-pointer hover:bg-surface-2" : ""
  } ${selected ? "bg-layer-1" : ""}`;
  const percentage = total > 0 ? Math.round((completed / total) * 100) : 0;
  const content = (
    <>
      <span className="w-4/6">{title}</span>
      <span className="flex w-2/6 items-center justify-end gap-1 px-2">
        <span className="flex h-5 items-center justify-center gap-1">
          <span className="w-8 text-right">{percentage}%</span>
        </span>
        <span>of {total}</span>
      </span>
    </>
  );
  return onClick ? (
    <button type="button" className={className} onClick={onClick} aria-pressed={selected}>
      {content}
    </button>
  ) : (
    <div className={className}>{content}</div>
  );
}
