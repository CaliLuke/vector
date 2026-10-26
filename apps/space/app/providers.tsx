/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ThemeProvider } from "next-themes";
// components
import { TranslationProvider } from "@plane/i18n";
import { AppProgressBar } from "@/lib/b-progress";
import { InstanceProvider } from "@/lib/instance-provider";
import { QueryProvider } from "@/lib/query-provider";
import { ToastProvider } from "@/lib/toast-provider";

export function AppProviders({ children }: { children: React.ReactNode }) {
  return (
    <ThemeProvider themes={["light", "dark"]} defaultTheme="system" enableSystem>
      <QueryProvider>
        <AppProgressBar />
        <TranslationProvider>
          <ToastProvider>
            <InstanceProvider>{children}</InstanceProvider>
          </ToastProvider>
        </TranslationProvider>
      </QueryProvider>
    </ThemeProvider>
  );
}
