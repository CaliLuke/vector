/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IWorkspaceMemberInvitation } from "@plane/types";
import { useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { InvitationSelectionForm } from "@/components/workspace/invitation-selection-form";
import { useCurrentUser } from "@/hooks/use-current-user";
import { workspacesOptions } from "@/queries/workspaces";
import { refreshCurrentUserSettings } from "@/queries/current-user-settings";
import { CommonOnboardingHeader } from "../common";
import { captureIdentityScope, isIdentityScopeCurrent } from "@/queries/identity";

type Props = {
  invitations: IWorkspaceMemberInvitation[];
  handleNextStep: () => Promise<void>;
  handleCurrentViewChange: () => void;
};
export function WorkspaceJoinInvitesStep({ invitations, handleNextStep, handleCurrentViewChange }: Props) {
  const queryClient = useQueryClient();
  const { data: user } = useCurrentUser();
  const router = useRouter();
  const routeHref = router.state.location.href;
  const identityScope = captureIdentityScope(queryClient, user?.id);
  return (
    <div className="flex flex-col gap-10">
      <CommonOnboardingHeader title="Join invites or create a workspace" description="All your work — unified." />
      {invitations.length ? (
        <InvitationSelectionForm
          invitations={invitations}
          onJoined={async () => {
            const isCurrentFlow = () =>
              Boolean(
                user?.id &&
                isIdentityScopeCurrent(queryClient, identityScope) &&
                router.state.location.href === routeHref
              );
            if (!user?.id || !isCurrentFlow()) throw new Error("The account session or onboarding route changed.");
            await queryClient.fetchQuery({ ...workspacesOptions(user.id), staleTime: 0 });
            if (!isCurrentFlow()) throw new Error("The account session or onboarding route changed.");
            await refreshCurrentUserSettings(queryClient, user.id);
            if (!isCurrentFlow()) throw new Error("The account session or onboarding route changed.");
            await handleNextStep();
          }}
          onCancel={handleCurrentViewChange}
          submitLabel="Continue"
          cancelLabel="Create new workspace"
        />
      ) : (
        <p>No invitations found</p>
      )}
    </div>
  );
}
