/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { RouterProvider, createRouter } from "@tanstack/react-router";
import { startTransition, StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { ThemeProvider } from "next-themes";
import { routeTree } from "./routeTree.gen";
import { installLocalLogging } from "@plane/utils/local-logging";

import polyfills from "@/lib/polyfills";
import { isStaleAssetErrorMessage, recoverFromStaleAsset } from "@/lib/stale-asset-error";

void polyfills;

// Production-only: in dev these errors come from the dev server itself (restarts,
// stale optimized deps) and auto-reloading would mask them.
if (import.meta.env.PROD) {
  window.addEventListener("vite:preloadError", (event) => {
    if (recoverFromStaleAsset()) event.preventDefault();
  });

  window.addEventListener("error", (event) => {
    if (isStaleAssetErrorMessage(event.message || "")) recoverFromStaleAsset();
  });

  window.addEventListener("unhandledrejection", (event) => {
    const reason = event.reason instanceof Error ? event.reason.message : String(event.reason ?? "");
    if (isStaleAssetErrorMessage(reason)) recoverFromStaleAsset();
  });
}

if (import.meta.env.DEV && import.meta.env.VITE_LOCAL_LOGGING === "1") {
  const stopLogging = installLocalLogging("plane-web-browser");
  import.meta.hot?.dispose(stopLogging);
}

// Optional session recording starts once with the application.
if (Number(process.env.VITE_ENABLE_SESSION_RECORDER) && process.env.VITE_SESSION_RECORDER_KEY) {
  const script = document.createElement("script");
  script.id = "clarity-tracking";
  script.textContent = `(function(c,l,a,r,i,t,y){
    c[a]=c[a]||function(){(c[a].q=c[a].q||[]).push(arguments)};
    t=l.createElement(r);t.async=1;t.src="https://www.clarity.ms/tag/"+i;
    y=l.getElementsByTagName(r)[0];if(y){y.parentNode.insertBefore(t,y);}
  })(window,document,"clarity","script",${JSON.stringify(process.env.VITE_SESSION_RECORDER_KEY)});`;
  document.head.appendChild(script);
}

const router = createRouter({
  routeTree,
  trailingSlash: "preserve",
  scrollRestoration: true,
  defaultPreload: "intent",
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

const root = document.getElementById("root");
if (!root) throw new Error("Missing application root");

startTransition(() => {
  createRoot(root).render(
    <StrictMode>
      <ThemeProvider themes={["light", "dark", "light-contrast", "dark-contrast", "custom"]} defaultTheme="system">
        <RouterProvider router={router} />
      </ThemeProvider>
    </StrictMode>
  );
});
