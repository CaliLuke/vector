/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef, useState } from "react";
// plane imports
import { EUserPermissions, MEMBER_TRACKER_ELEMENTS } from "@plane/constants";
import type { IUserLite } from "@plane/types";
import type { IMemberFilters } from "@/helpers/member-ordering";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { SearchIcon } from "@plane/propel/icons";
// components
import { MembersSettingsLoader } from "@/components/ui/loader/settings/members";
// hooks
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useWorkspaceMemberRoster, useProjectMemberRoster } from "@/hooks/use-member-rosters";
import type { RowData } from "@/components/project/settings/member-columns";
import type { IWorkspaceMember, TProjectMembership } from "@plane/types";
// local imports
import { MemberListFiltersDropdown } from "./dropdowns/filters/member-list";
import { ProjectMemberListItem } from "./member-list-item";
import { SendProjectInvitationModal } from "./send-project-invitation-modal";
import { sortProjectMembers } from "@/helpers/member-ordering";

type TProjectMemberListProps = {
  projectId: string;
  workspaceSlug: string;
};

function getProjectMemberDetails(
  memberships: TProjectMembership[],
  membersById: Map<string, IWorkspaceMember>,
  filters: IMemberFilters,
  searchQuery: string
): RowData[] {
  const visibleMemberships = memberships.filter((membership) => membersById.has(membership.member));
  const memberDetailsById = Object.fromEntries(
    visibleMemberships.map((membership) => {
      const member = membersById.get(membership.member)!.member;
      return [membership.member, { ...member, joining_date: membership.created_at ?? undefined }];
    })
  ) as Record<string, IUserLite>;
  const normalizedSearch = searchQuery.toLowerCase();
  return sortProjectMembers(visibleMemberships, memberDetailsById, (membership) => membership.member, filters).flatMap(
    (membership) => {
      const workspaceMember = membersById.get(membership.member);
      if (!workspaceMember) return [];
      const name = workspaceMember.member;
      const searchable = `${name.first_name} ${name.last_name} ${name.display_name}`.toLowerCase();
      if (!searchable.includes(normalizedSearch)) return [];
      return [
        {
          ...membership,
          member: { ...name, joining_date: membership.created_at ?? undefined },
          workspaceRole: workspaceMember.role,
        },
      ];
    }
  );
}

export function ProjectMemberList(props: TProjectMemberListProps) {
  const scopeKey = JSON.stringify([props.workspaceSlug, props.projectId]);
  return <ProjectMemberListView key={scopeKey} {...props} />;
}

function ProjectMemberListView(props: TProjectMemberListProps) {
  const { projectId, workspaceSlug } = props;
  // states
  const [inviteModal, setInviteModal] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const searchInput = useRef<HTMLInputElement | null>(null);
  const [memberFilters, setMemberFilters] = useState<IMemberFilters>({});
  const permissions = useProjectPermissions(workspaceSlug, projectId);
  const { projectRole, can: canProject } = permissions;
  const projectRoster = useProjectMemberRoster(workspaceSlug, projectId);
  const workspaceRoster = useWorkspaceMemberRoster(workspaceSlug);

  const { t } = useTranslation();

  useEffect(() => {
    searchInput.current?.focus();
  }, []);

  const memberDetails =
    projectRoster.memberships && workspaceRoster.membersById
      ? getProjectMemberDetails(projectRoster.memberships, workspaceRoster.membersById, memberFilters, searchQuery)
      : [];

  const isAdmin = canProject([EUserPermissions.ADMIN]);
  const updateFilters = (update: Partial<IMemberFilters>) => setMemberFilters((current) => ({ ...current, ...update }));

  // Handler for role filter updates
  const handleRoleFilterUpdate = (role: string) => {
    if (projectId) {
      const currentRoles = memberFilters.roles || [];
      const updatedRoles = currentRoles.includes(role)
        ? currentRoles.filter((r) => r !== role)
        : [...currentRoles, role];

      updateFilters({ roles: updatedRoles.length > 0 ? updatedRoles : undefined });
    }
  };

  const handleDisplayFilterUpdate = (update: Partial<IMemberFilters>) => updateFilters(update);
  const appliedRoleFilters = memberFilters.roles || [];

  return (
    <>
      <SendProjectInvitationModal
        isOpen={inviteModal}
        onClose={() => setInviteModal(false)}
        projectId={projectId}
        workspaceSlug={workspaceSlug}
      />
      <div className="flex items-center justify-between gap-4 overflow-x-hidden border-b border-subtle py-2">
        <div className="text-14 font-semibold">{t("common.members")}</div>
        <div className="flex items-center gap-2">
          <div className="flex items-center justify-start gap-1.5 rounded-md border border-subtle bg-surface-1 px-2 py-1">
            <SearchIcon className="h-3.5 w-3.5" />
            <input
              ref={searchInput}
              className="w-full max-w-[234px] border-none bg-transparent text-13 placeholder:text-placeholder focus:outline-none"
              placeholder="Search"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>
          <MemberListFiltersDropdown
            appliedFilters={appliedRoleFilters}
            handleUpdate={handleRoleFilterUpdate}
            memberType="project"
          />
          {isAdmin && (
            <Button
              variant="primary"
              onClick={() => {
                setInviteModal(true);
              }}
              data-ph-element={MEMBER_TRACKER_ELEMENTS.HEADER_ADD_BUTTON}
            >
              {t("add_member")}
            </Button>
          )}
        </div>
      </div>
      {permissions.isError ||
      projectRoster.query.isError ||
      workspaceRoster.query.isError ||
      (!permissions.isPending && projectRole === undefined) ? (
        <div role="alert" className="p-4 text-13 text-danger-primary">
          Unable to load project members. Please retry.
          <button
            type="button"
            className="ml-2 underline"
            onClick={() => {
              void permissions.membershipQuery.refetch();
              void permissions.rolesQuery.refetch();
              void projectRoster.query.refetch();
              void workspaceRoster.query.refetch();
            }}
          >
            Retry
          </button>
        </div>
      ) : permissions.isPending || projectRoster.query.isPending || workspaceRoster.query.isPending ? (
        <MembersSettingsLoader />
      ) : (
        <div className="divide-y divide-subtle overflow-scroll">
          {memberDetails.length !== 0 && (
            <ProjectMemberListItem
              memberDetails={memberDetails}
              projectId={projectId}
              workspaceSlug={workspaceSlug}
              displayFilters={memberFilters}
              handleDisplayFilterUpdate={handleDisplayFilterUpdate}
            />
          )}
          {memberDetails.length === 0 && (
            <h4 className="mt-16 text-center text-13 text-placeholder">{t("no_matching_members")}</h4>
          )}
        </div>
      )}
    </>
  );
}
