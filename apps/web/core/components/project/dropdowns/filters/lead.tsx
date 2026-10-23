/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import {
  WorkspaceMemberFilter,
  type WorkspaceMemberFilterProps,
} from "@/components/core/filters/workspace-member-filter";

type Props = Omit<WorkspaceMemberFilterProps, "title">;

export function FilterLead(props: Props) {
  return <WorkspaceMemberFilter {...props} title="Lead" />;
}
