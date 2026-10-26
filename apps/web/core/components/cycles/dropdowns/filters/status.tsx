/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { CYCLE_STATUS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { CatalogStatusFilter } from "@/components/core/filters/catalog-status-filter";
import type { CatalogStatusFilterProps } from "@/components/core/filters/catalog-status-filter";

export const FilterStatus = function FilterStatus(props: CatalogStatusFilterProps) {
  const { t } = useTranslation();

  return (
    <CatalogStatusFilter
      {...props}
      title="Status of the cycle"
      options={CYCLE_STATUS.map((status) => ({ value: status.value, title: t(status.i18n_title) }))}
    />
  );
};
