/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { QueryClientProvider } from "@tanstack/react-query";
import { createAppQueryClient } from "@plane/hooks";
import { useState } from "react";
export function QueryProvider({ children }: React.PropsWithChildren) {
  const [client] = useState(() => createAppQueryClient({ queries: { retry: 1, refetchOnWindowFocus: false } }));
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}
