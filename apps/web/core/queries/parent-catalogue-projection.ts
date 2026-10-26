/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import type { QueryClient } from "@tanstack/react-query";
import type { ICycle, IModule } from "@plane/types";
import { hashKey } from "@tanstack/react-query";
import { parentCatalogueKeys } from "./cycle-modules";
import type { ParentDomain, ParentCatalogueCollection } from "./cycle-modules";
import { captureParentAccess } from "./cycle-module-scope";
import type { ParentCatalogueScope } from "./cycle-module-scope";
export type ParentCatalogueProjection<T> = {
  scopeKey: string;
  rows: readonly T[] | undefined;
  isCurrent: () => boolean;
};
export type ParentNameProjections = {
  cycles: ParentCatalogueProjection<ICycle>;
  modules: ParentCatalogueProjection<IModule>;
};
export function captureParentCatalogueProjection<T>(
  client: QueryClient,
  domain: ParentDomain,
  scope: ParentCatalogueScope,
  rows: readonly T[] | undefined,
  collection: ParentCatalogueCollection = "active"
): ParentCatalogueProjection<T> {
  const key = parentCatalogueKeys.list(domain, scope, collection),
    query = client.getQueryCache().find({ queryKey: key, exact: true }),
    access = captureParentAccess(client, scope);
  return {
    scopeKey: hashKey(key),
    rows,
    isCurrent: () =>
      rows !== undefined &&
      access.isVerified() &&
      query?.state.status === "success" &&
      query.state.data === rows &&
      client.getQueryCache().find({ queryKey: key, exact: true }) === query,
  };
}
