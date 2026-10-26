/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
// types
import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { SearchIcon } from "@plane/propel/icons";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IWorkspaceBulkInviteFormData } from "@plane/types";
import { cn } from "@plane/utils/common";
// components
import { NotAuthorizedView } from "@/components/auth-screens/not-authorized-view";
import { CountChip } from "@/components/common/count-chip";
import { PageHead } from "@/components/core/page-title";
import { MemberListFiltersDropdown } from "@/components/project/dropdowns/filters/member-list";
import { WorkspaceMembersList } from "@/components/workspace/settings/members-list";
import { SettingsContentWrapper } from "@/components/settings/content-wrapper";
import { SendWorkspaceInvitationModal } from "@/components/workspace/members";
// hooks
import { useCurrentWorkspace } from "@/hooks/use-workspaces";
import { useWorkspacePermissions } from "@/hooks/use-permissions";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { useInviteWorkspaceMembers } from "@/hooks/use-member-roster-mutations";
import type { IMemberFilters } from "@/helpers/member-ordering";
// local imports
import { MembersWorkspaceSettingsHeader } from "./header";

function WorkspaceMembersSettingsPage(): React.JSX.Element | null {
  // router
  const { workspaceSlug } = Route.useParams();
  const [filtersByWorkspace, setFiltersByWorkspace] = useState<Record<string, IMemberFilters>>({});
  const filters = filtersByWorkspace[workspaceSlug] ?? {};

  const handleRoleFilterUpdate = (role: string) => {
    setFiltersByWorkspace((current) => {
      const currentFilters = current[workspaceSlug] ?? {};
      const currentRoles = currentFilters.roles ?? [];
      const updatedRoles = currentRoles.includes(role)
        ? currentRoles.filter((currentRole) => currentRole !== role)
        : [...currentRoles, role];
      return {
        ...current,
        [workspaceSlug]: { ...currentFilters, roles: updatedRoles.length > 0 ? updatedRoles : undefined },
      };
    });
  };

  const handleDisplayFilterUpdate = (updates: Partial<IMemberFilters>) => {
    setFiltersByWorkspace((current) => {
      const currentFilters = current[workspaceSlug];
      return {
        ...current,
        [workspaceSlug]: { ...currentFilters, ...updates },
      };
    });
  };

  return (
    <WorkspaceMembersSettingsContent
      key={workspaceSlug}
      workspaceSlug={workspaceSlug}
      filters={filters}
      onRoleFilterUpdate={handleRoleFilterUpdate}
      onDisplayFilterUpdate={handleDisplayFilterUpdate}
    />
  );
}

function WorkspaceMembersSettingsContent(props: {
  workspaceSlug: string;
  filters: IMemberFilters;
  onRoleFilterUpdate: (role: string) => void;
  onDisplayFilterUpdate: (updates: Partial<IMemberFilters>) => void;
}): React.JSX.Element | null {
  // Invitation and search drafts are owned by this workspace-keyed subtree.
  const { workspaceSlug, filters, onRoleFilterUpdate, onDisplayFilterUpdate } = props;
  const [inviteModal, setInviteModal] = useState(false);
  const [searchQuery, setSearchQuery] = useState<string>("");
  const permissions = useWorkspacePermissions(workspaceSlug);
  const { can: canWorkspace, isPending, isError, membershipQuery } = permissions;
  const roster = useWorkspaceMemberRoster(workspaceSlug);
  const inviteMembers = useInviteWorkspaceMembers(workspaceSlug);
  const { data: currentWorkspace } = useCurrentWorkspace();
  const { t } = useTranslation();
  const workspaceMemberCount = roster.members?.filter((member) => !member.member.is_bot).length ?? 0;

  // derived values
  const canPerformWorkspaceAdminActions = canWorkspace([EUserPermissions.ADMIN]);
  const canPerformWorkspaceMemberActions = canWorkspace([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  const handleWorkspaceInvite = async (data: IWorkspaceBulkInviteFormData) => {
    try {
      await inviteMembers.mutateAsync(data);

      setInviteModal(false);

      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Success!",
        message: t("workspace_settings.settings.members.invitations_sent_successfully"),
      });
    } catch (error: unknown) {
      let message = undefined;
      if (error instanceof Error) {
        const err = error as Error & { error?: string };
        message = err.error;
      }
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: `${message ?? t("something_went_wrong_please_try_again")}`,
      });

      throw error;
    }
  };

  // derived values
  const pageTitle = currentWorkspace?.name ? `${currentWorkspace.name} - Members` : undefined;
  const appliedRoleFilters = filters.roles || [];

  // if user is not authorized to view this page
  if (isPending) return null;

  if (isError) {
    return (
      <div className="flex items-center justify-center gap-3 p-8" role="alert">
        <span>{t("something_went_wrong_please_try_again")}</span>
        <Button variant="secondary" size="lg" onClick={() => void membershipQuery.refetch()}>
          {t("retry")}
        </Button>
      </div>
    );
  }

  if (!canPerformWorkspaceMemberActions) {
    return <NotAuthorizedView section="settings" className="h-auto" />;
  }

  return (
    <SettingsContentWrapper header={<MembersWorkspaceSettingsHeader />} hugging>
      <PageHead title={pageTitle} />
      <SendWorkspaceInvitationModal
        isOpen={inviteModal}
        onClose={() => setInviteModal(false)}
        onSubmit={handleWorkspaceInvite}
      />
      <section
        className={cn("size-full", {
          "opacity-60": !canPerformWorkspaceMemberActions,
        })}
      >
        <div className="flex items-center justify-between gap-4 pb-3.5">
          <h4 className="flex items-center gap-2.5 text-h3-medium">
            {t("workspace_settings.settings.members.title")}
            {workspaceMemberCount > 0 && <CountChip count={workspaceMemberCount} className="m-auto h-5" />}
          </h4>
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-1.5 rounded-md border border-subtle bg-surface-1 px-2.5 py-1.5">
              <SearchIcon className="h-3.5 w-3.5 text-placeholder" />
              <input
                className="w-full max-w-[234px] border-none bg-transparent text-body-xs-regular outline-none placeholder:text-placeholder"
                placeholder={`${t("search")}...`}
                value={searchQuery}
                // eslint-disable-next-line jsx-a11y/no-autofocus
                autoFocus
                onChange={(e) => setSearchQuery(e.target.value)}
              />
            </div>
            <MemberListFiltersDropdown
              appliedFilters={appliedRoleFilters}
              handleUpdate={onRoleFilterUpdate}
              memberType="workspace"
            />
            {canPerformWorkspaceAdminActions && (
              <Button variant="primary" size="lg" onClick={() => setInviteModal(true)}>
                {t("workspace_settings.settings.members.add_member")}
              </Button>
            )}
          </div>
        </div>
        <WorkspaceMembersList
          searchQuery={searchQuery}
          isAdmin={canPerformWorkspaceAdminActions}
          filters={filters}
          onFilterUpdate={onDisplayFilterUpdate}
        />
      </section>
    </SettingsContentWrapper>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/settings/_workspace/members")({
  component: WorkspaceMembersSettingsPage,
});
