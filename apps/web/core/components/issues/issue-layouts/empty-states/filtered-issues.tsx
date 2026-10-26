/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTranslation } from "@plane/i18n";
import { EmptyStateDetailed } from "@plane/propel/empty-state";

type Props = {
  clearFilters: (() => void | Promise<void>) | undefined;
  canClearFilters: boolean;
  clearFiltersLabel: string;
};

export function FilteredIssuesEmptyState({ clearFilters, canClearFilters, clearFiltersLabel }: Props) {
  const { t } = useTranslation();

  return (
    <EmptyStateDetailed
      assetKey="search"
      title={t("common_empty_state.search.title")}
      description={t("common_empty_state.search.description")}
      actions={[
        {
          label: clearFiltersLabel,
          onClick: () => {
            void clearFilters?.();
          },
          disabled: !canClearFilters || !clearFilters,
          variant: "secondary",
        },
      ]}
    />
  );
}
