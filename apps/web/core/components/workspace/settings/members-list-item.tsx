/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRouter } from "@tanstack/react-router";
import { useQueryClient } from "@tanstack/react-query";
import isEmpty from "lodash-es/isEmpty.js";
// plane imports
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IWorkspace, IWorkspaceMember, IUserSettings } from "@plane/types";
import { Table } from "@plane/ui";
// components
import { MembersLayoutLoader } from "@/components/ui/loader/layouts/members-layout-loader";
import { ConfirmWorkspaceMemberRemove } from "@/components/workspace/confirm-workspace-member-remove";
import type { RowData } from "@/components/workspace/settings/member-columns";
// hooks
import { useCurrentUser } from "@/hooks/use-current-user";
import { useCurrentUserSettings } from "@/hooks/use-current-user-settings";
import { useWorkspaces } from "@/hooks/use-workspaces";
import { useLeaveWorkspace } from "@/hooks/use-permissions";
import { useRemoveWorkspaceMember } from "@/hooks/use-member-roster-mutations";
import type { IMemberFilters } from "@/helpers/member-ordering";
import { refreshCurrentUserSettings } from "@/queries/current-user-settings";
import { captureIdentityScope, isIdentityScopeCurrent } from "@/queries/identity";
import { getWorkspaceDestination, workspaceKeys, workspacesOptions } from "@/queries/workspaces";
import type { Query, QueryClient } from "@tanstack/react-query";

// plane web imports
import { useMemberColumns } from "@/components/workspace/settings/useMemberColumns";

type Props = {
  memberDetails: IWorkspaceMember[];
  filters: IMemberFilters;
  onFilterUpdate: (updates: Partial<IMemberFilters>) => void;
};

type ScopeCheck = () => boolean;

async function removeLeftWorkspaceFromCatalogue(
  queryClient: QueryClient,
  userId: string,
  workspaceSlug: string,
  expectedQuery: Query | undefined,
  isCurrentIdentity: ScopeCheck
) {
  const queryKey = workspaceKeys.list(userId);
  if (
    !isCurrentIdentity() ||
    !expectedQuery ||
    queryClient.getQueryCache().find({ queryKey, exact: true }) !== expectedQuery
  )
    return;
  await queryClient.cancelQueries({ queryKey, exact: true });
  if (
    !isCurrentIdentity() ||
    !expectedQuery ||
    queryClient.getQueryCache().find({ queryKey, exact: true }) !== expectedQuery
  )
    return;
  queryClient.setQueryData<IWorkspace[]>(queryKey, (cached) =>
    cached?.filter((workspace) => workspace.slug !== workspaceSlug)
  );
}

async function navigateAfterWorkspaceLeave(
  navigate: (href: string) => Promise<unknown>,
  destination: string,
  isCurrentRoute: ScopeCheck
) {
  try {
    await navigate(destination);
  } catch {
    if (isCurrentRoute()) {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Workspace left",
        message: "Unable to choose a destination. Please reload the page.",
      });
    }
  }
}

async function continueAfterWorkspaceLeave({
  queryClient,
  userId,
  workspaceSlug,
  workspaceQuery,
  initialSettings,
  initialWorkspaces,
  isCurrentIdentity,
  isCurrentRoute,
  navigate,
}: {
  queryClient: QueryClient;
  userId: string;
  workspaceSlug: string;
  workspaceQuery: Query | undefined;
  initialSettings: IUserSettings | undefined;
  initialWorkspaces: IWorkspace[];
  isCurrentIdentity: ScopeCheck;
  isCurrentRoute: ScopeCheck;
  navigate: (href: string) => Promise<unknown>;
}) {
  if (!isCurrentIdentity()) return;
  await removeLeftWorkspaceFromCatalogue(queryClient, userId, workspaceSlug, workspaceQuery, isCurrentIdentity);
  if (!isCurrentRoute()) return;

  let settings = initialSettings?.id === userId ? initialSettings : undefined;
  const cachedWorkspaces = queryClient.getQueryData<IWorkspace[]>(workspaceKeys.list(userId));
  let remainingWorkspaces = (cachedWorkspaces ?? initialWorkspaces).filter(
    (workspace) => workspace.slug !== workspaceSlug
  );
  const [settingsResult, workspacesResult] = await Promise.allSettled([
    refreshCurrentUserSettings(queryClient, userId),
    queryClient.fetchQuery({ ...workspacesOptions(userId), staleTime: 0 }),
  ]);
  if (settingsResult.status === "fulfilled") settings = settingsResult.value;
  if (workspacesResult.status === "fulfilled")
    remainingWorkspaces = workspacesResult.value.filter((workspace) => workspace.slug !== workspaceSlug);
  if (!isCurrentRoute()) return;
  if (settingsResult.status === "rejected" || workspacesResult.status === "rejected") {
    setToast({
      type: TOAST_TYPE.ERROR,
      title: "Workspace left",
      message: "Unable to refresh your workspace list. Please reload the page.",
    });
  }
  if (isCurrentRoute()) {
    await navigateAfterWorkspaceLeave(navigate, getWorkspaceDestination(remainingWorkspaces, settings), isCurrentRoute);
  }
}

export function WorkspaceMembersListItem(props: Props) {
  const { memberDetails, filters, onFilterUpdate } = props;
  const { columns, workspaceSlug, removeMemberModal, setRemoveMemberModal } = useMemberColumns({
    filters,
    onFilterUpdate,
  });
  // router
  const router = useRouter();
  const queryClient = useQueryClient();
  // store hooks
  const { data: currentUser } = useCurrentUser();
  const { data: currentSettings } = useCurrentUserSettings();
  const { data: workspaces = [] } = useWorkspaces();
  const leaveWorkspace = useLeaveWorkspace(workspaceSlug ?? "");
  // derived values

  const handleLeaveWorkspace = async () => {
    if (!workspaceSlug || !currentUser?.id) throw new Error("A signed-in user and workspace are required.");
    const identityScope = captureIdentityScope(queryClient, currentUser.id);
    if (!identityScope) throw new Error("The account session changed before leaving the workspace.");
    const workspaceQuery = queryClient
      .getQueryCache()
      .find({ queryKey: workspaceKeys.list(currentUser.id), exact: true });
    const pathname = router.state.location.pathname;
    const isCurrentIdentity = () => isIdentityScopeCurrent(queryClient, identityScope);
    const isCurrentRoute = () => isCurrentIdentity() && router.state.location.pathname === pathname;

    // A failed primary write must reach the confirmation boundary so it stays open for retry.
    await leaveWorkspace.mutateAsync();
    await continueAfterWorkspaceLeave({
      queryClient,
      userId: currentUser.id,
      workspaceSlug,
      workspaceQuery,
      initialSettings: currentSettings,
      initialWorkspaces: workspaces,
      isCurrentIdentity,
      isCurrentRoute,
      navigate: (href) => router.navigate({ href }),
    });
  };

  // is the member current logged in user
  // const isCurrentUser = memberDetails?.member.id === currentUser?.id;
  // is the current logged in user admin
  // role change access-
  // 1. user cannot change their own role
  // 2. only admin or member can change role
  // 3. user cannot change role of higher role

  if (isEmpty(columns)) return <MembersLayoutLoader />;

  return (
    <div className="grid border-t border-subtle">
      {removeMemberModal &&
        (removeMemberModal.member.id === currentUser?.id ? (
          <ConfirmWorkspaceMemberRemove
            isOpen
            onClose={() => setRemoveMemberModal(null)}
            userDetails={{
              id: removeMemberModal.member.id,
              display_name: removeMemberModal.member.display_name || "",
            }}
            onSubmit={handleLeaveWorkspace}
          />
        ) : (
          <RemoveWorkspaceMemberConfirmation
            key={removeMemberModal.member.id}
            workspaceSlug={workspaceSlug ?? ""}
            membershipId={removeMemberModal.id}
            memberUserId={removeMemberModal.member.id}
            displayName={removeMemberModal.member.display_name || ""}
            onClose={() => setRemoveMemberModal(null)}
          />
        ))}
      <Table<RowData>
        columns={columns ?? []}
        data={memberDetails}
        keyExtractor={(rowData) => rowData.member.id}
        tHeadClassName="border-b border-subtle"
        thClassName="text-left font-medium divide-x-0 text-placeholder"
        tBodyClassName="divide-y-0"
        tBodyTrClassName="divide-x-0 p-4 h-10 text-secondary"
        tHeadTrClassName="divide-x-0"
      />
    </div>
  );
}

function RemoveWorkspaceMemberConfirmation(props: {
  workspaceSlug: string;
  membershipId: string;
  memberUserId: string;
  displayName: string;
  onClose: () => void;
}) {
  const { workspaceSlug, membershipId, memberUserId, displayName, onClose } = props;
  const removal = useRemoveWorkspaceMember(workspaceSlug, membershipId, memberUserId);
  return (
    <ConfirmWorkspaceMemberRemove
      isOpen
      onClose={onClose}
      userDetails={{ id: memberUserId, display_name: displayName }}
      onSubmit={() => removal.mutateAsync()}
    />
  );
}
