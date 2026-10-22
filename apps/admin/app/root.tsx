/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useState } from "react";
import type { ReactNode } from "react";
import { Links, Meta, Outlet, Scripts } from "react-router";
import type { LinksFunction } from "react-router";
import appleTouchIcon from "@/app/assets/vector.svg?url";
import favicon16 from "@/app/assets/vector.svg?url";
import favicon32 from "@/app/assets/vector.svg?url";
import faviconIco from "@/app/assets/vector.svg?url";
import { LogoSpinner } from "@/components/common/logo-spinner";
import globalStyles from "@/styles/globals.css?url";
import { AppProviders } from "@/providers";
import type { Route } from "./+types/root";
// fonts
// oxlint-disable-next-line import/no-unassigned-import -- The font package registers its CSS as a side effect.
import "@fontsource-variable/inter";
import interVariableWoff2 from "@fontsource-variable/inter/files/inter-latin-wght-normal.woff2?url";
// oxlint-disable-next-line import/no-unassigned-import -- The font package registers its CSS as a side effect.
import "@fontsource/material-symbols-rounded";
// oxlint-disable-next-line import/no-unassigned-import -- The font package registers its CSS as a side effect.
import "@fontsource/ibm-plex-mono";

const APP_TITLE = "Vector | Simple, extensible, open-source project management tool.";
const APP_DESCRIPTION =
  "Open-source project management tool to manage work items, sprints, and product roadmaps with peace of mind.";

export const links: LinksFunction = () => [
  { rel: "apple-touch-icon", sizes: "180x180", href: appleTouchIcon },
  { rel: "icon", type: "image/svg+xml", sizes: "any", href: favicon32 },
  { rel: "icon", type: "image/svg+xml", sizes: "any", href: favicon16 },
  { rel: "shortcut icon", href: faviconIco },
  { rel: "manifest", href: `/site.webmanifest.json` },
  { rel: "stylesheet", href: globalStyles },
  {
    rel: "preload",
    href: interVariableWoff2,
    as: "font",
    type: "font/woff2",
    crossOrigin: "anonymous",
  },
];

export function Layout({ children }: { children: ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <Meta />
        <Links />
      </head>
      <body className="antialiased" suppressHydrationWarning>
        <AppProviders>{children}</AppProviders>
        <Scripts />
      </body>
    </html>
  );
}

export const meta: Route.MetaFunction = () => [
  { title: APP_TITLE },
  { name: "description", content: APP_DESCRIPTION },
  { property: "og:title", content: APP_TITLE },
  { property: "og:description", content: APP_DESCRIPTION },
  { property: "og:url", content: "https://github.com/CaliLuke/vector" },
  {
    name: "keywords",
    content:
      "software development, customer feedback, software, accelerate, code management, release management, project management, work items tracking, agile, scrum, kanban, collaboration",
  },
  { name: "twitter:site", content: "Vector" },
];

export default function Root() {
  return (
    <div className="min-h-screen bg-canvas">
      <Outlet />
    </div>
  );
}

export function HydrateFallback() {
  const [isMounted, setIsMounted] = useState(false);

  useEffect(() => setIsMounted(true), []);

  if (!isMounted) return <div />;

  return (
    <div className="relative flex h-screen w-full items-center justify-center">
      <LogoSpinner />
    </div>
  );
}

export function ErrorBoundary({ error: _error }: Route.ErrorBoundaryProps) {
  return (
    <div>
      <p>Something went wrong.</p>
    </div>
  );
}
