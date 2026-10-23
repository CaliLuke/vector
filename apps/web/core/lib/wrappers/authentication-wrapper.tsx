/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRouterState, Navigate } from "@tanstack/react-router";
import type { ReactNode } from "react";
import { observer } from "mobx-react";

import useSWR from "swr";
// components
import { LogoSpinner } from "@/components/common/logo-spinner";
// helpers
import { EPageTypes } from "@/helpers/authentication.helper";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUser, useUserProfile, useUserSettings } from "@/hooks/store/user";

type TPageType = EPageTypes;

type TAuthenticationWrapper = {
  children: ReactNode;
  pageType?: TPageType;
};

const isValidURL = (url: string): boolean => {
  return url.startsWith("/") && !url.startsWith("//") && !url.includes("\\");
};

export const AuthenticationWrapper = observer(function AuthenticationWrapper(props: TAuthenticationWrapper) {
  // Keep redirects tied to the rendered page while the next route is loading.
  const location = useRouterState({ select: (state) => state.resolvedLocation ?? state.location });
  const currentHref = location.href;
  const searchParams = new URLSearchParams(location.searchStr);
  const nextPath = searchParams.get("next_path");
  // props
  const { children, pageType = EPageTypes.AUTHENTICATED } = props;
  // hooks
  const { isLoading: isUserLoading, data: currentUser, fetchCurrentUser } = useUser();
  const { data: currentUserProfile } = useUserProfile();
  const { data: currentUserSettings } = useUserSettings();
  const { loader: workspacesLoader, workspaces } = useWorkspace();

  const { isLoading: isUserSWRLoading } = useSWR("USER_INFORMATION", async () => await fetchCurrentUser(), {
    revalidateOnFocus: false,
    shouldRetryOnError: false,
  });

  const isUserOnboard =
    currentUserProfile?.is_onboarded ||
    (currentUserProfile?.onboarding_step?.profile_complete &&
      currentUserProfile?.onboarding_step?.workspace_create &&
      currentUserProfile?.onboarding_step?.workspace_invite &&
      currentUserProfile?.onboarding_step?.workspace_join) ||
    false;

  const getWorkspaceRedirectionUrl = (): string => {
    let redirectionRoute = "/create-workspace";

    // validating the nextPath from the router query
    if (nextPath && isValidURL(nextPath.toString())) {
      redirectionRoute = nextPath.toString();
      return redirectionRoute;
    }

    // validate the last and fallback workspace_slug
    const currentWorkspaceSlug =
      currentUserSettings?.workspace?.last_workspace_slug || currentUserSettings?.workspace?.fallback_workspace_slug;

    // validate the current workspace_slug is available in the user's workspace list
    const isCurrentWorkspaceValid = Object.values(workspaces || {}).findIndex(
      (workspace) => workspace.slug === currentWorkspaceSlug
    );

    if (isCurrentWorkspaceValid >= 0) redirectionRoute = `/${currentWorkspaceSlug}`;

    return redirectionRoute;
  };

  if ((isUserSWRLoading || isUserLoading || workspacesLoader) && !currentUser?.id)
    return (
      <div className="relative flex h-screen w-full items-center justify-center">
        <LogoSpinner />
      </div>
    );

  if (pageType === EPageTypes.PUBLIC) return <>{children}</>;

  if (!currentUser?.id) {
    if (pageType === EPageTypes.NON_AUTHENTICATED) return <>{children}</>;
    return <Navigate to="/" search={{ next_path: currentHref }} />;
  }

  const canEnterWorkspace = Boolean(currentUserProfile?.id && isUserOnboard);
  if (pageType === EPageTypes.AUTHENTICATED) return canEnterWorkspace ? <>{children}</> : <Navigate to="/onboarding" />;

  if (pageType === EPageTypes.NON_AUTHENTICATED)
    return <Navigate to={canEnterWorkspace ? getWorkspaceRedirectionUrl() : "/onboarding"} />;

  if (pageType === EPageTypes.ONBOARDING && canEnterWorkspace)
    return <Navigate to={getWorkspaceRedirectionUrl()} replace />;

  if (pageType === EPageTypes.SET_PASSWORD && !currentUser.is_password_autoset && canEnterWorkspace)
    return <Navigate to={getWorkspaceRedirectionUrl()} />;

  return <>{children}</>;
});
