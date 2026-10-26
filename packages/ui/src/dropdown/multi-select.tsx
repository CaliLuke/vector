/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { DropdownControl } from "./control";
import type { IMultiSelectDropdown } from "./dropdown";

export function MultiSelectDropdown(props: IMultiSelectDropdown) {
  return <DropdownControl {...props} multiple={true} />;
}
