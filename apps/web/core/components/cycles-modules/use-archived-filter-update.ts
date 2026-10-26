/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback } from "react";
import { toggleCatalogueFilterValues } from "./toggle-catalogue-filter-values";

type ArchivedFilters<TKey extends string> = Partial<Record<TKey, string[] | null>>;

export function useArchivedFilterUpdate<TKey extends string>(
  projectId: string | undefined,
  currentFilters: ArchivedFilters<TKey> | undefined,
  update: (patch: { archivedFilters: ArchivedFilters<TKey> }) => void
) {
  return useCallback(
    (key: TKey, value: string | string[]) => {
      if (!projectId) return;
      const nextValues = toggleCatalogueFilterValues(currentFilters?.[key], value);
      update({ archivedFilters: { ...currentFilters, [key]: nextValues } });
    },
    [currentFilters, projectId, update]
  );
}
