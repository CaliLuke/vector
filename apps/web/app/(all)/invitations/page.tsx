/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
import { createFileRoute, Link, useRouter } from "@tanstack/react-router";
import { useRef } from "react";

import { useQueryClient } from "@tanstack/react-query";
// plane imports
import { useTranslation } from "@plane/i18n";
// types
import { PlaneLogo } from "@plane/propel/icons";
// assets
import emptyInvitation from "@/app/assets/empty-state/invitation.svg?url";
// components
import { InvitationSelectionForm } from "@/components/workspace/invitation-selection-form";
import { EmptyState } from "@/components/common/empty-state";
// hooks
import { useCurrentUser } from "@/hooks/use-current-user";
import { useUpdateUserProfile } from "@/hooks/use-current-user-profile";
import { workspacesOptions } from "@/queries/workspaces";
import { captureIdentityScope, isIdentityScopeCurrent } from "@/queries/identity";

// services
import { AuthenticationWrapper } from "@/lib/wrappers/authentication-wrapper";
// plane web services
import { WorkspaceService } from "@/services/workspace.service";

const workspaceService = new WorkspaceService();

function UserInvitationsPage(): React.JSX.Element | null {
  const queryClient = useQueryClient();
  // router
  const router = useRouter();
  // store hooks
  const { t } = useTranslation();
  const { data: currentUser } = useCurrentUser();
  const { mutateAsync: updateUserProfile } = useUpdateUserProfile();
  const identityScope = captureIdentityScope(queryClient, currentUser?.id);
  const routeHref = router.state.location.href;
  const lastWorkspacePreferenceSaved = useRef<string | undefined>(undefined);

  const { data: invitations } = useRecordCollection({
    queryKey: ["USER_WORKSPACE_INVITATIONS"],
    queryFn: () => workspaceService.userWorkspaceInvitations(),
  });

  const finishJoining = async (ids: string[]) => {
    const isCurrentFlow = () =>
      isIdentityScopeCurrent(queryClient, identityScope) && router.state.location.href === routeHref;
    if (!isCurrentFlow()) throw new Error("The account session or invitations route changed.");
    const destination = invitations?.find((invitation) => invitation.id === ids[0])?.workspace;
    if (destination && currentUser?.id) {
      const savedPreference = `${currentUser.id}:${destination.id}`;
      if (lastWorkspacePreferenceSaved.current !== savedPreference) {
        await updateUserProfile({ last_workspace_id: destination.id });
        if (!isCurrentFlow()) throw new Error("The account session or invitations route changed.");
        lastWorkspacePreferenceSaved.current = savedPreference;
      }
      if (!isCurrentFlow()) throw new Error("The account session or invitations route changed.");
      await queryClient.fetchQuery({ ...workspacesOptions(currentUser.id), staleTime: 0 });
    }
    if (!isCurrentFlow()) throw new Error("The account session or invitations route changed.");
    await router.navigate({ href: destination ? `/${destination.slug}` : "/" });
  };

  return (
    <AuthenticationWrapper>
      <div className="flex h-full flex-col gap-y-2 overflow-hidden sm:flex-row sm:gap-y-0">
        <div className="relative h-1/6 flex-shrink-0 sm:w-2/12 md:w-3/12 lg:w-1/5">
          <div className="absolute top-1/2 left-0 h-[0.5px] w-full -translate-y-1/2 border-b-[0.5px] border-subtle sm:top-0 sm:left-1/2 sm:h-screen sm:w-[0.5px] sm:-translate-x-1/2 sm:translate-y-0 sm:border-r-[0.5px] md:left-1/3" />
          <Link
            to="/"
            className="absolute top-1/2 left-5 z-10 grid -translate-y-1/2 place-items-center px-3 sm:top-12 sm:left-1/2 sm:-translate-x-[15px] sm:translate-y-0 sm:px-0 sm:py-5 md:left-1/3"
          >
            <PlaneLogo className="h-9 w-auto text-primary" />
          </Link>
          <div className="absolute top-1/4 right-4 -translate-y-1/2 text-13 text-primary sm:fixed sm:top-12 sm:right-16 sm:translate-y-0 sm:py-5">
            {currentUser?.email}
          </div>
        </div>
        {invitations ? (
          invitations.length > 0 ? (
            <div className="relative flex h-full justify-center px-8 pb-8 sm:w-10/12 sm:items-center sm:justify-start sm:p-0 sm:pr-[8.33%] md:w-9/12 lg:w-4/5">
              <div className="w-full space-y-10">
                <h5 className="text-16">{t("we_see_that_someone_has_invited_you_to_join_a_workspace")}</h5>
                <h4 className="text-20 font-semibold">{t("join_a_workspace")}</h4>
                <InvitationSelectionForm
                  invitations={invitations}
                  onJoined={finishJoining}
                  onCancel={() => void router.navigate({ href: "/" })}
                  submitLabel={t("accept_and_join")}
                  cancelLabel={t("go_home")}
                />
              </div>
            </div>
          ) : (
            <div className="fixed top-0 left-0 grid h-full w-full place-items-center">
              <EmptyState
                title={t("no_pending_invites")}
                description={t("you_can_see_here_if_someone_invites_you_to_a_workspace")}
                image={emptyInvitation}
                primaryButton={{
                  text: t("back_to_home"),
                  onClick: () => void router.navigate({ href: "/" }),
                }}
              />
            </div>
          )
        ) : null}
      </div>
    </AuthenticationWrapper>
  );
}

export const Route = createFileRoute("/invitations/")({
  component: UserInvitationsPage,
});
