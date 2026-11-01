/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { CycleIcon } from "@plane/propel/icons";
import { PlanningActivity } from "./planning";
import type { PlanningActivityProps } from "./planning";

export function IssueCycleActivity(props: PlanningActivityProps) {
  return (
    <PlanningActivity {...props} kind="cycle" icon={<CycleIcon className="h-4 w-4 flex-shrink-0 text-secondary" />} />
  );
}
