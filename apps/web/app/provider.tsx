/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { lazy, Suspense } from "react";
import { useTheme } from "next-themes";
// Vector Imports
import { TranslationProvider } from "@plane/i18n";
import { Toast } from "@plane/propel/toast";
// helpers
import { resolveGeneralTheme } from "@plane/utils/theme-legacy";
// application query and UI state
import { AppStateProvider } from "@/providers/app-state";

// lazy imports
const AppProgressBar = lazy(function AppProgressBar() {
  return import("@/lib/b-progress/AppProgressBar");
});

const ProfileEffects = lazy(function ProfileEffects() {
  return import("@/lib/wrappers/profile-effects");
});

const InstanceWrapper = lazy(function InstanceWrapper() {
  return import("@/lib/wrappers/instance-wrapper");
});

export interface IAppProvider {
  children: React.ReactNode;
}

export function AppProvider(props: IAppProvider) {
  const { children } = props;
  // themes
  const { resolvedTheme } = useTheme();

  return (
    <AppStateProvider>
      <Suspense fallback={null}>
        <>
          <AppProgressBar />
          <TranslationProvider>
            <Toast theme={resolveGeneralTheme(resolvedTheme)} />
            <ProfileEffects>
              <InstanceWrapper>
                <Suspense>{children}</Suspense>
              </InstanceWrapper>
            </ProfileEffects>
          </TranslationProvider>
        </>
      </Suspense>
    </AppStateProvider>
  );
}
