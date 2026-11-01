/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Spinner as SharedSpinner } from "@plane/propel/spinners";
import type { ISpinner } from "@plane/propel/spinners";
import { cn } from "../utils";

export type { ISpinner } from "@plane/propel/spinners";

export function Spinner({ className, ...props }: ISpinner) {
  return <SharedSpinner {...props} className={cn("fill-accent-primary", className)} indicatorClassName="" />;
}
