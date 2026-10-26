/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { queryOptions, skipToken } from "@tanstack/react-query";
import {
  SitesStateService,
  SitesLabelService,
  SitesMemberService,
  SitesModuleService,
  SitesCycleService,
} from "@plane/services";
import { sortStates } from "@/helpers/state.helper";

const statesService = new SitesStateService();
export function statesOptions(anchor: string | undefined) {
  return queryOptions({
    queryKey: ["public", anchor, "states"],
    queryFn: anchor ? ({ signal }) => statesService.list(anchor, signal) : skipToken,
    staleTime: 30_000,
    select: (states) => sortStates([...states]),
  });
}

const labelsService = new SitesLabelService();
export function labelsOptions(anchor: string | undefined) {
  return queryOptions({
    queryKey: ["public", anchor, "labels"],
    queryFn: anchor ? ({ signal }) => labelsService.list(anchor, signal) : skipToken,
    staleTime: 30_000,
  });
}

const membersService = new SitesMemberService();
export function membersOptions(anchor: string | undefined) {
  return queryOptions({
    queryKey: ["public", anchor, "members"],
    queryFn: anchor ? ({ signal }) => membersService.list(anchor, signal) : skipToken,
    staleTime: 30_000,
  });
}

const modulesService = new SitesModuleService();
export function modulesOptions(anchor: string | undefined) {
  return queryOptions({
    queryKey: ["public", anchor, "modules"],
    queryFn: anchor ? ({ signal }) => modulesService.list(anchor, signal) : skipToken,
    staleTime: 30_000,
  });
}

const cyclesService = new SitesCycleService();
export function cyclesOptions(anchor: string | undefined) {
  return queryOptions({
    queryKey: ["public", anchor, "cycles"],
    queryFn: anchor ? ({ signal }) => cyclesService.list(anchor, signal) : skipToken,
    staleTime: 30_000,
  });
}
