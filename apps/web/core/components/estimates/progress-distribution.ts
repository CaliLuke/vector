/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import type { ICycle, IModule } from "@plane/types";

/** Both progress charts select canonical distributions using the same estimate mode. */
export function progressDistribution(parent: ICycle | IModule | null, mode: string) {
  return mode === "points" ? parent?.estimate_distribution : parent?.distribution || undefined;
}

export function progressCount(points: number | undefined, issues: number | undefined, estimateMode: boolean) {
  return (estimateMode ? points : issues) || 0;
}
