/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { QueryClientProvider } from "@tanstack/react-query";
import { createAppQueryClient } from "@plane/hooks";
import { useState } from "react";
import type { ReactNode } from "react";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { CycleModulePreferencesProvider } from "./cycle-module-preferences";
import { NativeCommandStateProvider } from "./native-command-state";
import { CalendarStateProvider } from "./issue-layout-ui-state";
import { SubWorkItemPreferencesProvider } from "./sub-work-item-preferences";
import { GanttViewPreferenceProvider } from "./gantt-view-preferences";
import { SidebarStateProvider } from "./sidebar-state";
import { IssueDetailUiStateProvider } from "./issue-detail-ui-state";
import { IntakePreferencesProvider } from "./intake-preferences";

export function AppStateProvider({ children }: { children: ReactNode }) {
  const [client] = useState(() => createAppQueryClient({ queries: { staleTime: 2_000, retry: 3 } }));
  const generation = useIdentityGeneration(client);

  return (
    <QueryClientProvider key={generation} client={client}>
      <IssueDetailUiStateProvider>
        <GanttViewPreferenceProvider>
          <CalendarStateProvider>
            <SidebarStateProvider>
              <SubWorkItemPreferencesProvider>
                <CycleModulePreferencesProvider>
                  <IntakePreferencesProvider>
                    <NativeCommandStateProvider>{children}</NativeCommandStateProvider>
                  </IntakePreferencesProvider>
                </CycleModulePreferencesProvider>
              </SubWorkItemPreferencesProvider>
            </SidebarStateProvider>
          </CalendarStateProvider>
        </GanttViewPreferenceProvider>
      </IssueDetailUiStateProvider>
    </QueryClientProvider>
  );
}
