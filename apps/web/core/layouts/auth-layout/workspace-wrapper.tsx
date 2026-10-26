/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link, useParams } from "@tanstack/react-router";
import type { ReactNode } from "react";

// ui
import { LogOut } from "lucide-react";
import { Button, getButtonStyling } from "@plane/propel/button";
import { PlaneLogo } from "@plane/propel/icons";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { Tooltip } from "@plane/propel/tooltip";
import { cn } from "@plane/utils/common";
// assets
import WorkSpaceNotAvailable from "@/app/assets/workspace/workspace-not-available.png?url";
// components
import { LogoSpinner } from "@/components/common/logo-spinner";
// constants
// hooks
import { useWorkspaceProjects } from "@/hooks/use-projects";
import { useWorkspaceStates } from "@/hooks/use-project-states";
import { useWorkspaceProjectRoles } from "@/hooks/use-permissions";
import { useCurrentUser, useSignOut } from "@/hooks/use-current-user";
import { useWorkspaces } from "@/hooks/use-workspaces";
import { usePlatformOS } from "@/hooks/use-platform-os";
import { useFavorites } from "@/hooks/use-favorites";

interface IWorkspaceAuthWrapper {
  children: ReactNode;
  isLoading?: boolean;
}

type TWorkspaceLoadState = "workspace-error" | "loading" | "ready";

function getWorkspaceLoadState(
  isParentLoading: boolean,
  isWorkspacesPending: boolean,
  hasWorkspaceData: boolean,
  hasWorkspaceError: boolean,
  hasCurrentWorkspace: boolean,
  isPermissionPending: boolean
): TWorkspaceLoadState {
  if (hasWorkspaceError && !hasWorkspaceData) return "workspace-error";
  if (isParentLoading || isWorkspacesPending || !hasWorkspaceData || (hasCurrentWorkspace && isPermissionPending)) {
    return "loading";
  }
  return "ready";
}

export function WorkspaceAuthWrapper(props: IWorkspaceAuthWrapper) {
  const { children, isLoading: isParentLoading = false } = props;
  // router params
  const { workspaceSlug } = useParams({ strict: false });
  // store hooks
  const workspacesQuery = useWorkspaces();
  // derived values
  const allWorkspaces = workspacesQuery.data;
  const workspaces = allWorkspaces ?? [];
  const currentWorkspace = allWorkspaces?.find((workspace) => workspace.slug === workspaceSlug);
  const permissions = useWorkspaceProjectRoles(currentWorkspace?.slug);
  const accessibleWorkspaceSlug = permissions.workspaceRole !== undefined ? currentWorkspace?.slug : undefined;
  useWorkspaceEntities(accessibleWorkspaceSlug);
  const loadState = getWorkspaceLoadState(
    isParentLoading,
    workspacesQuery.isPending,
    allWorkspaces !== undefined,
    workspacesQuery.isError,
    currentWorkspace !== undefined,
    permissions.isPending
  );

  if (loadState === "workspace-error") {
    return (
      <div role="alert" className="flex h-full flex-col items-center justify-center gap-3">
        <p>Unable to load your workspaces. Please try again.</p>
        <Button disabled={workspacesQuery.isFetching} onClick={() => void workspacesQuery.refetch()}>
          Retry
        </Button>
      </div>
    );
  }

  if (loadState === "loading") {
    return (
      <div className="grid h-full place-items-center rounded-lg border border-subtle p-4">
        <div className="flex flex-col items-center gap-3 text-center">
          <LogoSpinner />
        </div>
      </div>
    );
  }

  if (currentWorkspace === undefined) return <WorkspaceNotFound hasWorkspaces={workspaces.length > 0} />;

  if (permissions.isError) {
    return (
      <div role="alert" className="flex h-full flex-col items-center justify-center gap-3">
        <p>Unable to load your workspace access. Please try again.</p>
        <Button
          disabled={permissions.membershipQuery.isFetching || permissions.rolesQuery.isFetching}
          onClick={() => {
            if (permissions.membershipQuery.isError) void permissions.membershipQuery.refetch();
            else void permissions.rolesQuery.refetch();
          }}
        >
          Retry
        </Button>
      </div>
    );
  }

  // while user does not have access to view that workspace
  if (permissions.workspaceRole === undefined) {
    return (
      <div className={`h-screen w-full overflow-hidden bg-surface-1`}>
        <div className="grid h-full place-items-center p-4">
          <div className="space-y-8 text-center">
            <div className="space-y-2">
              <h3 className="text-16 font-semibold">Not Authorized!</h3>
              <p className="mx-auto w-1/2 text-13 text-secondary">
                You{"'"}re not a member of this workspace. Please contact the workspace admin to get an invitation or
                check your pending invitations.
              </p>
            </div>
            <div className="flex items-center justify-center gap-2">
              <Link to="/invitations/">
                <span>
                  <Button variant="secondary">Check pending invites</Button>
                </span>
              </Link>
              <Link to="/create-workspace/">
                <span>
                  <Button variant="primary">Create new workspace</Button>
                </span>
              </Link>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return <>{children}</>;
}

function WorkspaceNotFound({ hasWorkspaces }: { hasWorkspaces: boolean }) {
  const { data: currentUser } = useCurrentUser();
  const { mutateAsync: signOut } = useSignOut();
  const { isMobile } = usePlatformOS();
  const handleSignOut = async () => {
    await signOut().catch(() =>
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Failed to sign out. Please try again.",
      })
    );
  };

  return (
    <div className="relative flex h-full w-full flex-col items-center justify-center bg-surface-2">
      <div className="relative container mx-auto flex h-full w-full flex-col overflow-hidden overflow-y-auto px-5 py-14 md:px-0">
        <div className="relative flex flex-shrink-0 items-center justify-between gap-4">
          <div className="z-10 flex-shrink-0 bg-surface-2 py-4">
            <PlaneLogo className="h-9 w-auto text-primary" />
          </div>
          <div className="relative flex items-center gap-2">
            <div className="text-13 font-medium">{currentUser?.email}</div>
            <button
              type="button"
              aria-label="Sign out"
              className="relative flex h-6 w-6 flex-shrink-0 cursor-pointer items-center justify-center overflow-hidden rounded-sm hover:bg-layer-1"
              onClick={() => void handleSignOut()}
            >
              <Tooltip tooltipContent={"Sign out"} position="top" className="ml-2" isMobile={isMobile}>
                <LogOut size={14} />
              </Tooltip>
            </button>
          </div>
        </div>
        <div className="relative flex h-full w-full flex-grow flex-col items-center justify-center space-y-3">
          <div className="relative flex-shrink-0">
            <img src={WorkSpaceNotAvailable} className="h-[220px] object-contain object-center" alt="Vector logo" />
          </div>
          <h3 className="text-center text-16 font-semibold">Workspace not found</h3>
          <p className="text-center text-13 text-secondary">
            No workspace found with the URL. It may not exist or you lack authorization to view it.
          </p>
          <div className="flex items-center justify-center gap-2 pt-4">
            {hasWorkspaces && (
              <Link to="/" className={cn(getButtonStyling("primary", "base"))}>
                Go Home
              </Link>
            )}
            {hasWorkspaces && (
              <Link
                to="/settings/profile/$profileTabId/"
                params={{ profileTabId: "general" }}
                className={cn(getButtonStyling("secondary", "base"))}
              >
                Visit Profile
              </Link>
            )}
            {!hasWorkspaces && (
              <Link to="/create-workspace/" className={cn(getButtonStyling("secondary", "base"))}>
                Create new workspace
              </Link>
            )}
          </div>
        </div>

        <div className="absolute top-0 bottom-0 left-4 w-0 bg-layer-1 md:w-0.5" />
      </div>
    </div>
  );
}

function useWorkspaceEntities(accessibleWorkspaceSlug: string | undefined) {
  useWorkspaceProjects(accessibleWorkspaceSlug);
  useFavorites(accessibleWorkspaceSlug);
  useWorkspaceStates(accessibleWorkspaceSlug);
}
