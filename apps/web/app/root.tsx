/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useState } from "react";
import type { ReactNode } from "react";
import Script from "next/script";
import { Links, Meta, Outlet, Scripts } from "react-router";
import type { LinksFunction } from "react-router";
import { ThemeProvider, useTheme } from "next-themes";
// plane imports
import { SITE_DESCRIPTION, SITE_NAME } from "@plane/constants";
import { cn } from "@plane/utils";
// types
// assets
import favicon16 from "@/app/assets/vector.svg?url";
import favicon32 from "@/app/assets/vector.svg?url";
import faviconIco from "@/app/assets/vector.svg?url";
import icon180 from "@/app/assets/icons/icon-180x180.png?url";
import icon512 from "@/app/assets/icons/icon-512x512.png?url";
import ogImage from "@/app/assets/og-image.png?url";
import globalStyles from "@/styles/globals.css?url";
import type { Route } from "./+types/root";
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

export const links: LinksFunction = () => [
  { rel: "icon", type: "image/svg+xml", sizes: "any", href: favicon32 },
  { rel: "icon", type: "image/svg+xml", sizes: "any", href: favicon16 },
  { rel: "shortcut icon", href: faviconIco },
  { rel: "manifest", href: "/site.webmanifest.json" },
  { rel: "apple-touch-icon", href: icon512 },
  { rel: "apple-touch-icon", sizes: "180x180", href: icon180 },
  { rel: "apple-touch-icon", sizes: "512x512", href: icon512 },
  { rel: "manifest", href: "/manifest.json" },
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
  const isSessionRecorderEnabled = parseInt(process.env.VITE_ENABLE_SESSION_RECORDER || "0");

  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <meta name="theme-color" content="#fff" />
        {/* Meta info for PWA */}
        <meta name="application-name" content="Vector" />
        <meta name="apple-mobile-web-app-capable" content="yes" />
        <meta name="apple-mobile-web-app-status-bar-style" content="default" />
        <meta name="apple-mobile-web-app-title" content={SITE_NAME} />
        <meta name="format-detection" content="telephone=no" />
        <meta name="mobile-web-app-capable" content="yes" />
        <Meta />
        <Links />
      </head>
      <body suppressHydrationWarning>
        <div id="context-menu-portal" />
        <div id="editor-portal" />
        <ThemeProvider themes={["light", "dark", "light-contrast", "dark-contrast", "custom"]} defaultTheme="system">
          {children}
        </ThemeProvider>
        <Scripts />
        {!!isSessionRecorderEnabled && process.env.VITE_SESSION_RECORDER_KEY && (
          <Script id="clarity-tracking">
            {`(function(c,l,a,r,i,t,y){
              c[a]=c[a]||function(){(c[a].q=c[a].q||[]).push(arguments)};
              t=l.createElement(r);t.async=1;t.src="https://www.clarity.ms/tag/"+i;
              y=l.getElementsByTagName(r)[0];if(y){y.parentNode.insertBefore(t,y);}
          })(window, document, "clarity", "script", "${process.env.VITE_SESSION_RECORDER_KEY}");`}
          </Script>
        )}
      </body>
    </html>
  );
}

export const meta: Route.MetaFunction = () => [
  { title: APP_TITLE },
  { name: "description", content: SITE_DESCRIPTION },
  { property: "og:title", content: APP_TITLE },
  {
    property: "og:description",
    content: "Open-source project management tool to manage work items, cycles, and product roadmaps easily",
  },
  { property: "og:url", content: "https://github.com/CaliLuke/vector" },
  { property: "og:image", content: ogImage },
  { property: "og:image:width", content: "1200" },
  { property: "og:image:height", content: "630" },
  { property: "og:image:alt", content: "Vector - Modern project management" },
  {
    name: "keywords",
    content:
      "software development, plan, ship, software, accelerate, code management, release management, project management, work item tracking, agile, scrum, kanban, collaboration",
  },
  { name: "twitter:site", content: "Vector" },
  { name: "twitter:card", content: "summary_large_image" },
  { name: "twitter:image", content: ogImage },
  { name: "twitter:image:width", content: "1200" },
  { name: "twitter:image:height", content: "630" },
  { name: "twitter:image:alt", content: "Vector - Modern project management" },
];

export default function Root() {
  return (
    <AppProvider>
      <div className={cn("relative flex h-screen w-full flex-col overflow-hidden bg-canvas", "desktop-app-container")}>
        <main className="relative h-full w-full overflow-hidden">
          <Outlet />
        </main>
      </div>
    </AppProvider>
  );
}

export function HydrateFallback() {
  const [isMounted, setIsMounted] = useState(false);
  const { resolvedTheme } = useTheme();

  useEffect(() => setIsMounted(true), []);

  if (!isMounted || resolvedTheme === undefined) return <div />;

  return (
    <div className="relative flex h-screen w-full items-center justify-center bg-canvas">
      <LogoSpinner />
    </div>
  );
}

export function ErrorBoundary({ error }: Route.ErrorBoundaryProps) {
  // A stale chunk failure surfaces here as React Router's own wrapper error
  // (the failed dynamic import itself never reaches a window event) — recover
  // the same way entry.client.tsx does instead of just showing the error page.
  if (import.meta.env.PROD && isStaleAssetError(error)) recoverFromStaleAsset();

  return <CustomErrorComponent error={error} />;
}
