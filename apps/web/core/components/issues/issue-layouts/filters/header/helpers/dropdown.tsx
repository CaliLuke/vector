/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ComponentProps } from "react";
import { FiltersDropdown as SharedFiltersDropdown } from "@plane/ui/dropdowns";

export function FiltersDropdown(props: ComponentProps<typeof SharedFiltersDropdown>) {
  return <SharedFiltersDropdown {...props} responsive />;
}
