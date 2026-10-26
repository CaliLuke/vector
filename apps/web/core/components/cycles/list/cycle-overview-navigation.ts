/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { generateQueryParams } from "@plane/utils/router";

export function getCycleOverviewHref(pathname: string, searchParams: URLSearchParams, cycleId: string) {
  const query = generateQueryParams(searchParams, ["peekCycle"]);
  const nextSearch = searchParams.get("peekCycle") === cycleId ? query : `${query && `${query}&`}peekCycle=${cycleId}`;
  return `${pathname}?${nextSearch}`;
}
