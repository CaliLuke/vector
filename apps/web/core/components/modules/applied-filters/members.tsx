/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { AppliedMembersFiltersProps } from "@/components/common/applied-filters/members";
import { AppliedMembersFilters as MemberFilters } from "@/components/common/applied-filters/members";

export function AppliedMembersFilters(props: AppliedMembersFiltersProps) {
  return <MemberFilters {...props} className="p-1" />;
}
