/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TFilterConfig, TFilterConditionNode, TFilterProperty, TFilterValue } from "@plane/types";

export function filterConfigByProperty<P extends TFilterProperty>(
  configs: readonly TFilterConfig<P>[],
  property: P
): TFilterConfig<P> | undefined {
  return configs.find((config) => config.id === property);
}

export function availableFilterConfigs<P extends TFilterProperty>(
  configs: readonly TFilterConfig<P>[],
  conditions: readonly TFilterConditionNode<P, TFilterValue>[]
): TFilterConfig<P>[] {
  const appliedProperties = new Set(conditions.map((condition) => condition.property));
  return configs.filter(
    (config) => config.isEnabled && (config.allowMultipleFilters || !appliedProperties.has(config.id))
  );
}
