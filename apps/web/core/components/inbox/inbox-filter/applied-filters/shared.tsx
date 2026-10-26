/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { CloseIcon } from "@plane/propel/icons";
import { Tag } from "@plane/ui";

export type TAppliedFilterChip<TKey extends string = string> = {
  key: TKey;
  label: ReactNode;
  leading?: ReactNode;
  onRemove?: () => void;
};

export function InboxAppliedFilterGroup<TKey extends string>(props: {
  label: ReactNode;
  options: TAppliedFilterChip<TKey>[];
  hasFilters?: boolean;
  onClearAll?: () => void;
}) {
  if (!(props.hasFilters ?? props.options.length > 0)) return null;
  return (
    <Tag>
      <div className="text-11 text-secondary">{props.label}</div>
      {props.options.map(({ key, ...option }) => (
        <InboxAppliedFilterChip key={key} {...option} />
      ))}
      {props.onClearAll && <InboxAppliedFilterClear ariaLabel="Clear all filters" onClick={props.onClearAll} />}
    </Tag>
  );
}

function InboxAppliedFilterChip(props: Omit<TAppliedFilterChip, "key">) {
  return (
    <div className="relative flex items-center gap-1 rounded-sm bg-layer-1 p-1 text-11">
      {props.leading}
      <div className="truncate text-11">{props.label}</div>
      {props.onRemove && <InboxAppliedFilterClear ariaLabel="Remove filter" onClick={props.onRemove} />}
    </div>
  );
}

function InboxAppliedFilterClear(props: { ariaLabel: string; onClick: () => void }) {
  return (
    <button
      type="button"
      aria-label={props.ariaLabel}
      className="relative flex h-3 w-3 flex-shrink-0 cursor-pointer appearance-none items-center justify-center overflow-hidden border-0 bg-transparent p-0 text-tertiary transition-all hover:text-secondary"
      onClick={props.onClick}
    >
      <CloseIcon className="h-3 w-3" />
    </button>
  );
}

export function toggleAppliedFilterValue<T>(values: T[], value: T): T[] {
  return values.includes(value) ? values.filter((candidate) => candidate !== value) : [...values, value];
}
