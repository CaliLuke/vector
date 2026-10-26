/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { useTranslation } from "@plane/i18n";
import { FilterHeader } from "./filter-header";
import { FilterOption } from "./filter-option";
export type FilterSelectionProps = {
  appliedFilters: string[] | null;
  handleUpdate: (id: string) => void;
  searchQuery: string;
};
type Props = FilterSelectionProps & {
  title: string;
  options: { id: string; name: string; icon: React.ReactNode }[];
  loading?: boolean;
  error?: boolean;
  retry?: () => void;
};
export function FilterSection({
  title,
  options,
  appliedFilters,
  handleUpdate,
  searchQuery,
  loading,
  error,
  retry,
}: Props) {
  const [open, setOpen] = useState(true);
  const [expanded, setExpanded] = useState(false);
  const { t } = useTranslation();
  const filtered = options.filter((option) => option.name.toLowerCase().includes(searchQuery.toLowerCase()));
  const selected = appliedFilters ?? [];
  const visible = expanded ? filtered : filtered.slice(0, 5);
  return (
    <>
      <FilterHeader
        title={selected.length ? `${title} (${selected.length})` : title}
        isPreviewEnabled={open}
        handleIsPreviewEnabled={() => setOpen((value) => !value)}
      />
      {open && (
        <div>
          {loading ? (
            <p className="text-11 text-placeholder">Loading…</p>
          ) : error ? (
            <p role="alert">
              Could not load options. <button onClick={retry}>Retry</button>
            </p>
          ) : (
            <>
              {visible.map((option) => (
                <FilterOption
                  key={option.id}
                  title={option.name}
                  icon={option.icon}
                  isChecked={selected.includes(option.id)}
                  onClick={() => handleUpdate(option.id)}
                />
              ))}
              {!filtered.length && (
                <p className="text-11 text-placeholder italic">{t("common.search.no_matches_found")}</p>
              )}
              {filtered.length > 5 && (
                <button
                  type="button"
                  className="ml-8 text-11 font-medium text-accent-primary"
                  onClick={() => setExpanded((value) => !value)}
                >
                  {expanded ? "View less" : "View all"}
                </button>
              )}
            </>
          )}
        </div>
      )}
    </>
  );
}
