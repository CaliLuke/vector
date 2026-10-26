/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { reportLocalQueryError } from "@plane/utils/local-logging";
import { useState } from "react";
export function QueryProvider({ children }: React.PropsWithChildren) {
  const [client] = useState(
    () =>
      new QueryClient({
        queryCache: new QueryCache({ onError: (error) => reportLocalQueryError(error, "query") }),
        mutationCache: new MutationCache({ onError: (error) => reportLocalQueryError(error, "mutation") }),
        defaultOptions: { queries: { retry: 1, refetchOnWindowFocus: false } },
      })
  );
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}
