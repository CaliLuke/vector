/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TBaseFilterFieldConfig } from "./shared";
import type { TFilterValue } from "../expression";

export const EXTENDED_FILTER_FIELD_TYPE = {
  TEXT: "text",
  NUMBER: "number",
  NUMBER_RANGE: "number_range",
} as const;

export type TTextFilterFieldConfig = TBaseFilterFieldConfig & {
  type: typeof EXTENDED_FILTER_FIELD_TYPE.TEXT;
  defaultValue?: string;
};

export type TNumberFilterFieldConfig = TBaseFilterFieldConfig & {
  type: typeof EXTENDED_FILTER_FIELD_TYPE.NUMBER;
  defaultValue?: number;
};

export type TNumberRangeFilterFieldConfig = TBaseFilterFieldConfig & {
  type: typeof EXTENDED_FILTER_FIELD_TYPE.NUMBER_RANGE;
  defaultValue?: number[];
};

export type TLiteralFilterFieldConfig =
  | TTextFilterFieldConfig
  | TNumberFilterFieldConfig
  | TNumberRangeFilterFieldConfig;
export type TExtendedFilterFieldConfigs<_V extends TFilterValue = TFilterValue> = TLiteralFilterFieldConfig;
