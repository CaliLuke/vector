/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TWorkItemFilterProperty } from "@plane/types";
// components
import type { TWorkItemFilterEditor } from "./filters-hoc/shared";
import type { TFiltersRowProps } from "@/components/rich-filters/filters-row";
import { FiltersRow } from "@/components/rich-filters/filters-row";

type TWorkItemFiltersRowProps = Omit<TFiltersRowProps<TWorkItemFilterProperty>, "editor"> & {
  editor: TWorkItemFilterEditor;
};

export const WorkItemFiltersRow = function WorkItemFiltersRow(props: TWorkItemFiltersRowProps) {
  return <FiltersRow {...props} />;
};
