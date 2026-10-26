/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TSupportedOperators } from "@plane/types";
import { CORE_OPERATORS } from "@plane/types";

export type TFiltersOperatorConfigs = {
  allowedOperators: Set<TSupportedOperators>;
  allowNegative: boolean;
};

export type TUseFiltersOperatorConfigsProps = {
  workspaceSlug: string;
};

const filtersOperatorConfigs: TFiltersOperatorConfigs = {
  allowedOperators: new Set(Object.values(CORE_OPERATORS)),
  allowNegative: true,
};

export const useFiltersOperatorConfigs = (_props: TUseFiltersOperatorConfigsProps): TFiltersOperatorConfigs =>
  filtersOperatorConfigs;
