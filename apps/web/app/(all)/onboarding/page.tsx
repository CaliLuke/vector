/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute } from "@tanstack/react-router";

import { useQuery, skipToken } from "@tanstack/react-query";

// components
import { LogoSpinner } from "@/components/common/logo-spinner";
import { OnboardingRoot } from "@/components/onboarding";
// constants
// helpers
import { EPageTypes } from "@/helpers/authentication.helper";
// hooks
import { useCurrentUser } from "@/hooks/use-current-user";
// wrappers
import { AuthenticationWrapper } from "@/lib/wrappers/authentication-wrapper";
// services
import { WorkspaceService } from "@/services/workspace.service";

const workspaceService = new WorkspaceService();

function OnboardingPage(): React.JSX.Element | null {
  // store hooks
  const { data: user } = useCurrentUser();

  // fetching user workspace invitations
  const { isLoading: invitationsLoader, data: invitations } = useQuery({
    queryKey: ["USER_WORKSPACE_INVITATIONS_LIST", user?.id],
    queryFn: user?.id ? () => workspaceService.userWorkspaceInvitations() : skipToken,
  });

  return (
    <AuthenticationWrapper pageType={EPageTypes.ONBOARDING}>
      <div className="relative flex size-full overflow-hidden rounded-lg bg-canvas transition-all duration-300 ease-in-out">
        <div className="size-full flex-grow overflow-hidden p-2 transition-all duration-300 ease-in-out">
          <div className="shadow-md relative flex h-full w-full flex-col overflow-hidden rounded-lg border border-subtle bg-surface-1">
            {user && !invitationsLoader ? (
              <OnboardingRoot invitations={invitations ?? []} />
            ) : (
              <div className="grid h-full w-full place-items-center">
                <LogoSpinner />
              </div>
            )}
          </div>
        </div>
      </div>
    </AuthenticationWrapper>
  );
}

export const Route = createFileRoute("/onboarding/")({
  component: OnboardingPage,
});
