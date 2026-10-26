/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRouterState, Navigate } from "@tanstack/react-router";
import type { ReactNode } from "react";

// components
import { LogoSpinner } from "@/components/common/logo-spinner";
// helpers
import { EPageTypes } from "@/helpers/authentication.helper";
// hooks
import { useCurrentUser } from "@/hooks/use-current-user";
import { useCurrentUserProfile } from "@/hooks/use-current-user-profile";
import { useCurrentUserSettings } from "@/hooks/use-current-user-settings";
import { useWorkspaces } from "@/hooks/use-workspaces";
import { getWorkspaceDestination } from "@/queries/workspaces";
import { Button } from "@plane/propel/button";
import type { IWorkspace, IUserSettings, TUserProfile } from "@plane/types";
import type { IUser } from "@plane/types";

type TAuthenticationWrapper = {
  children: ReactNode;
  pageType?: EPageTypes;
};

type TRedirect = { to: string; search?: { next_path: string }; replace?: boolean };

type TAccountQueryGateProps = {
  isPending: boolean;
  isError: boolean;
  onRetry: () => void;
  children: ReactNode;
};

function AccountQueryGate({ isPending, isError, onRetry, children }: TAccountQueryGateProps) {
  if (isPending)
    return (
      <div className="relative flex h-screen w-full items-center justify-center">
        <LogoSpinner />
      </div>
    );

  if (isError)
    return (
      <div className="flex h-screen flex-col items-center justify-center gap-4" role="alert">
        <p>Unable to load your account. Please try again.</p>
        <Button onClick={onRetry}>Retry</Button>
      </div>
    );

  return <>{children}</>;
}

function isValidURL(url: string): boolean {
  return url.startsWith("/") && !url.startsWith("//") && !url.includes("\\");
}

function hasCompletedOnboarding(profile: TUserProfile | undefined): boolean {
  const steps = profile?.onboarding_step;
  return Boolean(
    profile?.is_onboarded ||
    (steps && steps.profile_complete && steps.workspace_create && steps.workspace_invite && steps.workspace_join)
  );
}

function getAuthenticationRedirect(
  pageType: EPageTypes,
  user: IUser | null | undefined,
  profile: TUserProfile | undefined,
  currentHref: string,
  nextPath: string | null,
  settings: IUserSettings | undefined,
  workspaces: IWorkspace[]
): TRedirect | undefined {
  if (pageType === EPageTypes.PUBLIC) return undefined;

  if (!user?.id)
    return pageType === EPageTypes.NON_AUTHENTICATED ? undefined : { to: "/", search: { next_path: currentHref } };

  return getSignedInRedirect(pageType, user, profile, nextPath, settings, workspaces);
}

function getSignedInRedirect(
  pageType: EPageTypes,
  user: IUser,
  profile: TUserProfile | undefined,
  nextPath: string | null,
  settings: IUserSettings | undefined,
  workspaces: IWorkspace[]
): TRedirect | undefined {
  const canEnterWorkspace = Boolean(profile?.id && hasCompletedOnboarding(profile));
  if (pageType === EPageTypes.AUTHENTICATED) return canEnterWorkspace ? undefined : { to: "/onboarding" };

  const destination = canEnterWorkspace
    ? nextPath && isValidURL(nextPath)
      ? nextPath
      : getWorkspaceDestination(workspaces, settings)
    : "/onboarding";
  if (pageType === EPageTypes.NON_AUTHENTICATED) return { to: destination };
  if (pageType === EPageTypes.ONBOARDING && canEnterWorkspace) return { to: destination, replace: true };
  if (pageType === EPageTypes.SET_PASSWORD && !user.is_password_autoset && canEnterWorkspace)
    return { to: destination };
  return undefined;
}

type TAuthenticationContentProps = {
  children: ReactNode;
  pageType: EPageTypes;
  user: IUser | null | undefined;
  profile: TUserProfile | undefined;
  currentHref: string;
  nextPath: string | null;
  settings: IUserSettings | undefined;
  workspaces: IWorkspace[];
};

function AuthenticationContent(props: TAuthenticationContentProps) {
  const redirect = getAuthenticationRedirect(
    props.pageType,
    props.user,
    props.profile,
    props.currentHref,
    props.nextPath,
    props.settings,
    props.workspaces
  );
  if (redirect?.search) return <Navigate to="/" search={redirect.search} />;
  if (redirect) return <Navigate to={redirect.to} replace={redirect.replace} />;
  return <>{props.children}</>;
}

export function AuthenticationWrapper(props: TAuthenticationWrapper) {
  const location = useRouterState({ select: (state) => state.resolvedLocation ?? state.location });
  const searchParams = new URLSearchParams(location.searchStr);
  const { children, pageType = EPageTypes.AUTHENTICATED } = props;
  const userQuery = useCurrentUser();
  const profileQuery = useCurrentUserProfile();
  const settingsQuery = useCurrentUserSettings();
  const workspacesQuery = useWorkspaces();
  const currentUser = userQuery.data;
  const accountQueries = currentUser?.id ? [userQuery, profileQuery, settingsQuery, workspacesQuery] : [userQuery];

  return (
    <AccountQueryGate
      isPending={accountQueries.some((query) => query.isPending)}
      isError={accountQueries.some((query) => query.isError)}
      onRetry={() => accountQueries.filter((query) => query.isError).forEach((query) => void query.refetch())}
    >
      <AuthenticationContent
        pageType={pageType}
        user={currentUser}
        profile={profileQuery.data}
        currentHref={location.href}
        nextPath={searchParams.get("next_path")}
        settings={settingsQuery.data}
        workspaces={workspacesQuery.data ?? []}
      >
        {children}
      </AuthenticationContent>
    </AccountQueryGate>
  );
}
