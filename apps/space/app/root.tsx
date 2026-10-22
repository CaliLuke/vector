/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Links, Meta, Outlet, Scripts } from "react-router";
// assets
import appleTouchIcon from "@/app/assets/vector.svg?url";
import favicon16 from "@/app/assets/vector.svg?url";
import favicon32 from "@/app/assets/vector.svg?url";
import faviconIco from "@/app/assets/vector.svg?url";
import siteWebmanifest from "@/app/assets/favicon/site.webmanifest?url";
import { LogoSpinner } from "@/components/common/logo-spinner";
import globalStyles from "@/styles/globals.css?url";
// types
import type { Route } from "./+types/root";
// local imports
import ErrorPage from "./error";
import { AppProviders } from "./providers";
// fonts
// oxlint-disable-next-line import/no-unassigned-import -- The font package registers its CSS as a side effect.
import "@fontsource-variable/inter";
import interVariableWoff2 from "@fontsource-variable/inter/files/inter-latin-wght-normal.woff2?url";
// oxlint-disable-next-line import/no-unassigned-import -- The font package registers its CSS as a side effect.
import "@fontsource/material-symbols-rounded";
// oxlint-disable-next-line import/no-unassigned-import -- The font package registers its CSS as a side effect.
import "@fontsource/ibm-plex-mono";

const APP_TITLE = "Vector Publish | Make your Vector boards public with one-click";
const APP_DESCRIPTION = "Public project boards and feedback for Vector";

export const links: Route.LinksFunction = () => [
  { rel: "apple-touch-icon", sizes: "180x180", href: appleTouchIcon },
  { rel: "icon", type: "image/svg+xml", sizes: "any", href: favicon32 },
  { rel: "icon", type: "image/svg+xml", sizes: "any", href: favicon16 },
  { rel: "shortcut icon", href: faviconIco },
  { rel: "manifest", href: siteWebmanifest },
  { rel: "stylesheet", href: globalStyles },
  {
    rel: "preload",
    href: interVariableWoff2,
    as: "font",
    type: "font/woff2",
    crossOrigin: "anonymous",
  },
];

export const headers: Route.HeadersFunction = () => ({
  "Referrer-Policy": "origin-when-cross-origin",
  "X-Content-Type-Options": "nosniff",
  "X-DNS-Prefetch-Control": "on",
  "Strict-Transport-Security": "max-age=31536000; includeSubDomains; preload",
});

export function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <meta name="robots" content="noindex, nofollow" />
        <Meta />
        <Links />
      </head>
      <body>
        <div id="editor-portal" />
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
      "software development, customer feedback, software, accelerate, code management, release management, project management, work item tracking, agile, scrum, kanban, collaboration",
  },
  { name: "twitter:site", content: "Vector" },
];

export default function Root() {
  return <Outlet />;
}

export function HydrateFallback() {
  return (
    <div className="relative flex h-screen w-full items-center justify-center bg-surface-1">
      <LogoSpinner />
    </div>
  );
}

export function ErrorBoundary({ error: _error }: Route.ErrorBoundaryProps) {
  return <ErrorPage />;
}
