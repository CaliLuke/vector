/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { createRootRoute, HeadContent, Outlet } from "@tanstack/react-router";
import { SITE_DESCRIPTION } from "@plane/constants";
import { cn } from "@plane/utils";
// types
// assets
import favicon16 from "@/app/assets/vector.svg?url";
import favicon32 from "@/app/assets/vector.svg?url";
import faviconIco from "@/app/assets/vector.svg?url";
import icon180 from "@/app/assets/icons/icon-180x180.png?url";
import icon512 from "@/app/assets/icons/icon-512x512.png?url";
import globalStyles from "@/styles/globals.css?url";
import PageNotFound from "./not-found";
// components
import { LogoSpinner } from "@/components/common/logo-spinner";
// lib
import { isStaleAssetError, recoverFromStaleAsset } from "@/lib/stale-asset-error";
// local
import { CustomErrorComponent } from "./error";
import { AppProvider } from "./provider";
// fonts
// oxlint-disable-next-line import/no-unassigned-import -- The font package registers its CSS as a side effect.
import "@fontsource-variable/inter";
import interVariableWoff2 from "@fontsource-variable/inter/files/inter-latin-wght-normal.woff2?url";
// oxlint-disable-next-line import/no-unassigned-import -- The font package registers its CSS as a side effect.
import "@fontsource/material-symbols-rounded";
// oxlint-disable-next-line import/no-unassigned-import -- The font package registers its CSS as a side effect.
import "@fontsource/ibm-plex-mono";

const APP_TITLE = "Vector | Simple, extensible, open-source project management tool.";

const links = () => [
  { rel: "icon", type: "image/svg+xml", sizes: "any", href: favicon32 },
  { rel: "icon", type: "image/svg+xml", sizes: "any", href: favicon16 },
  { rel: "shortcut icon", href: faviconIco },
  { rel: "manifest", href: "/manifest.json" },
  { rel: "apple-touch-icon", href: icon512 },
  { rel: "apple-touch-icon", sizes: "180x180", href: icon180 },
  { rel: "apple-touch-icon", sizes: "512x512", href: icon512 },
  { rel: "stylesheet", href: globalStyles },
  {
    rel: "preload",
    href: interVariableWoff2,
    as: "font",
    type: "font/woff2",
    crossOrigin: "anonymous" as const,
  },
];

const meta = () => [
  { title: APP_TITLE },
  { name: "description", content: SITE_DESCRIPTION },
  { property: "og:title", content: APP_TITLE },
  {
    property: "og:description",
    content: "Open-source project management tool to manage work items, cycles, and product roadmaps easily",
  },
  { property: "og:url", content: "https://github.com/CaliLuke/vector" },
  { property: "og:image", content: icon512 },
  { property: "og:image:width", content: "512" },
  { property: "og:image:height", content: "512" },
  { property: "og:image:alt", content: "Vector - Modern project management" },
  {
    name: "keywords",
    content:
      "software development, plan, ship, software, accelerate, code management, release management, project management, work item tracking, agile, scrum, kanban, collaboration",
  },
  { name: "twitter:site", content: "Vector" },
  { name: "twitter:card", content: "summary_large_image" },
  { name: "twitter:image", content: icon512 },
  { name: "twitter:image:width", content: "512" },
  { name: "twitter:image:height", content: "512" },
  { name: "twitter:image:alt", content: "Vector - Modern project management" },
];

function Root() {
  return (
    <>
      <HeadContent />
      <AppProvider>
        <div
          className={cn("relative flex h-screen w-full flex-col overflow-hidden bg-canvas", "desktop-app-container")}
        >
          <main className="relative h-full w-full overflow-hidden">
            <Outlet />
          </main>
        </div>
      </AppProvider>
    </>
  );
}

function PendingPage() {
  return (
    <div className="relative flex h-screen w-full items-center justify-center bg-canvas">
      <LogoSpinner />
    </div>
  );
}

function ErrorBoundary({ error }: { error: unknown }) {
  // A stale chunk failure surfaces here as the router's error boundary
  // (the failed dynamic import itself never reaches a window event) — recover
  // the same way entry.client.tsx does instead of just showing the error page.
  if (import.meta.env.PROD && isStaleAssetError(error)) recoverFromStaleAsset();

  return <CustomErrorComponent error={error} />;
}

export const Route = createRootRoute({
  component: Root,
  head: () => ({ meta: meta(), links: links() }),
  errorComponent: ErrorBoundary,
  notFoundComponent: PageNotFound,
  pendingComponent: PendingPage,
});
