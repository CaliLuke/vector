/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { startTransition, StrictMode } from "react";
import { hydrateRoot } from "react-dom/client";
import { HydratedRouter } from "react-router/dom";
import { installLocalLogging } from "@plane/utils";

if (import.meta.env.DEV && import.meta.env.VITE_LOCAL_LOGGING === "1") {
  const stopLogging = installLocalLogging("plane-admin-browser");
  import.meta.hot?.dispose(stopLogging);
}

startTransition(() => {
  hydrateRoot(
    document,
    <StrictMode>
      <HydratedRouter />
    </StrictMode>
  );
});
