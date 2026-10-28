/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useSelector } from "@tanstack/react-store";
import type { QueryClient } from "@tanstack/react-query";
import { identitySession } from "@/queries/identity";

export function useIdentityGeneration(client: QueryClient) {
  return useSelector(identitySession(client), (generation) => generation);
}
