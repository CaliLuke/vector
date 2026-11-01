/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { orderedFilterKeys } from "./keys";
import { useTranslation } from "@plane/i18n";
import type { ReactNode } from "react";
import { CloseIcon } from "@plane/propel/icons";
import { Tag } from "@plane/ui";
import { replaceUnderscoreIfSnakeCase } from "@plane/utils/string";

export function AppliedFilterTag({
  filterKey,
  value,
  onClear,
  valuesClassName,
  children,
}: {
  filterKey: string;
  value: unknown;
  onClear?: () => void;
  valuesClassName?: string;
  children: ReactNode;
}) {
  if (!value || (Array.isArray(value) && value.length === 0)) return null;
  const label = replaceUnderscoreIfSnakeCase(filterKey);
  const values = (
    <>
      {children}
      {onClear && <RemoveAppliedFilter filterKey={filterKey} onRemove={onClear} />}
    </>
  );
  return (
    <Tag>
      <span className="text-11 text-tertiary">{label}</span>
      {valuesClassName ? <div className={valuesClassName}>{values}</div> : values}
    </Tag>
  );
}

export function AppliedFiltersRow<K extends string>({
  filters,
  keys,
  onRemove,
  onClearAll,
  valuesClassName,
  children,
}: {
  filters: Partial<Record<K, unknown>>;
  keys: readonly K[];
  onRemove?: (key: K) => void;
  onClearAll?: () => void;
  valuesClassName?: string;
  children: (key: K) => ReactNode;
}) {
  if (!filters || Object.keys(filters).length === 0) return null;
  return (
    <div className="flex flex-wrap items-stretch gap-2 bg-surface-1">
      {orderedFilterKeys(filters, keys).map((key) => (
        <AppliedFilterTag
          key={key}
          filterKey={key}
          value={filters[key]}
          onClear={onRemove ? () => onRemove(key) : undefined}
          valuesClassName={valuesClassName}
        >
          {children(key)}
        </AppliedFilterTag>
      ))}
      {onClearAll && <ClearAppliedFilters onClear={onClearAll} />}
    </div>
  );
}

export function ClearAppliedFilters({ onClear }: { onClear: () => void }) {
  const { t } = useTranslation();
  return (
    <button type="button" onClick={onClear}>
      <Tag>
        {t("common.clear_all")}
        <CloseIcon height={12} width={12} strokeWidth={2} />
      </Tag>
    </button>
  );
}

export function RemoveAppliedFilter({ filterKey, onRemove }: { filterKey: string; onRemove: () => void }) {
  return (
    <button
      type="button"
      aria-label={`Remove ${replaceUnderscoreIfSnakeCase(filterKey)} filter`}
      className="grid place-items-center text-tertiary hover:text-secondary"
      onClick={onRemove}
    >
      <CloseIcon height={12} width={12} strokeWidth={2} />
    </button>
  );
}
