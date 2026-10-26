/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { MutationCache, QueryCache, QueryClient } from "@tanstack/react-query";
import type { QueryClientConfig } from "@tanstack/react-query";
import { reportLocalQueryError } from "@plane/utils/local-logging";

export function createAppQueryClient(defaultOptions: QueryClientConfig["defaultOptions"] = {}) {
  return new QueryClient({
    queryCache: new QueryCache({ onError: (error) => reportLocalQueryError(error, "query") }),
    mutationCache: new MutationCache({ onError: (error) => reportLocalQueryError(error, "mutation") }),
    defaultOptions,
  });
}
