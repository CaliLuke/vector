/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { Button } from "@plane/propel/button";
import type { IUser, IWorkspace } from "@plane/types";
import { CreateWorkspaceForm } from "@/components/workspace/create-workspace-form";
import { useQuery } from "@tanstack/react-query";
import { instanceConfigOptions } from "@/queries/instance";
import { useUpdateUserProfile } from "@/hooks/use-current-user-profile";
import { refreshCurrentUserSettings } from "@/queries/current-user-settings";
import { captureIdentityScope, isIdentityScopeCurrent } from "@/queries/identity";
import { CommonOnboardingHeader } from "../common";

type Props = {
  user: IUser | undefined;
  onComplete: (skipInvites?: boolean) => Promise<void>;
  handleCurrentViewChange: () => void;
  hasInvitations?: boolean;
};

export function WorkspaceCreateStep({ user, onComplete, handleCurrentViewChange, hasInvitations = false }: Props) {
  const { data: config } = useQuery(instanceConfigOptions);
  const queryClient = useQueryClient();
  const router = useRouter();
  const routeHref = router.state.location.href;
  const identityScope = captureIdentityScope(queryClient, user?.id);
  const updateUserProfile = useUpdateUserProfile();
  const complete = async (workspace: IWorkspace) => {
    const isCurrentFlow = () =>
      Boolean(user && isIdentityScopeCurrent(queryClient, identityScope) && router.state.location.href === routeHref);
    if (!user?.id || !isCurrentFlow()) throw new Error("The account session or onboarding route changed.");
    await updateUserProfile.mutateAsync({ last_workspace_id: workspace.id });
    if (!isCurrentFlow()) throw new Error("The account session or onboarding route changed.");
    await refreshCurrentUserSettings(queryClient, user.id);
    if (!isCurrentFlow()) throw new Error("The account session or onboarding route changed.");
    await onComplete(workspace.organization_size === "Just myself");
  };
  if (config?.is_workspace_creation_disabled)
    return (
      <p className="text-center text-14 text-tertiary">
        Your instance admin has restricted workspace creation. Ask a workspace owner or admin to invite you, then return
        to join.
      </p>
    );
  return (
    <div className="flex flex-col gap-10">
      <CommonOnboardingHeader title="Create your workspace" description="All your work — unified." />
      <CreateWorkspaceForm
        onSubmit={complete}
        secondaryButton={
          hasInvitations ? (
            <Button variant="ghost" type="button" onClick={handleCurrentViewChange}>
              Join existing workspace
            </Button>
          ) : undefined
        }
      />
    </div>
  );
}
