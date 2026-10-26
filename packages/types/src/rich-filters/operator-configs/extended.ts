/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type {
  TTextFilterFieldConfig,
  TNumberFilterFieldConfig,
  TNumberRangeFilterFieldConfig,
  TDateFilterFieldConfig,
  TSingleSelectFilterFieldConfig,
} from "../field-types";

// ----------------------------- EXACT Operator -----------------------------
export type TExtendedExactOperatorConfigs = TTextFilterFieldConfig | TNumberFilterFieldConfig;

// ----------------------------- IN Operator -----------------------------
export type TExtendedInOperatorConfigs = never;

// ----------------------------- RANGE Operator -----------------------------
export type TExtendedRangeOperatorConfigs = TNumberRangeFilterFieldConfig;

// ----------------------------- Extended Operator Specific Configs -----------------------------
export type TExtendedOperatorSpecificConfigs = {
  contains: TTextFilterFieldConfig;
  isnull: TSingleSelectFilterFieldConfig<boolean>;
  lt: TNumberFilterFieldConfig | TDateFilterFieldConfig<string>;
  lte: TNumberFilterFieldConfig | TDateFilterFieldConfig<string>;
  gt: TNumberFilterFieldConfig | TDateFilterFieldConfig<string>;
  gte: TNumberFilterFieldConfig | TDateFilterFieldConfig<string>;
};
