/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ThemeProvider } from "next-themes";
import { AppProgressBar } from "@/lib/b-progress";
// local imports
import { ToastWithTheme } from "./toast";
import { QueryProvider } from "./query";
import { SidebarProvider } from "./sidebar";

export function CoreProviders({ children }: { children: React.ReactNode }) {
  return (
    <ThemeProvider themes={["light", "dark"]} defaultTheme="system" enableSystem>
      <AppProgressBar />
      <ToastWithTheme />
      <QueryProvider>
        <SidebarProvider>{children}</SidebarProvider>
      </QueryProvider>
    </ThemeProvider>
  );
}
