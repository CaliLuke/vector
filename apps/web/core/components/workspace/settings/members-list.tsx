/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useState } from "react";
import sortBy from "lodash-es/sortBy.js";
import { Disclosure } from "@headlessui/react";
// plane imports
import { useTranslation } from "@plane/i18n";
import { ChevronDownIcon } from "@plane/propel/icons";
import { Collapsible } from "@plane/ui";
import { Button } from "@plane/propel/button";
import type { IWorkspaceMember, IWorkspaceMemberInvitation } from "@plane/types";
// components
import { CountChip } from "@/components/common/count-chip";
import { MembersSettingsLoader } from "@/components/ui/loader/settings/members";
// hooks
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { useWorkspaceInvitations } from "@/hooks/use-member-roster-mutations";
import type { IMemberFilters } from "@/helpers/member-ordering";
import { sortWorkspaceMembers } from "@/helpers/member-ordering";
// local imports
import { WorkspaceInvitationsListItem } from "./invitations-list-item";
import { WorkspaceMembersListItem } from "./members-list-item";

function QueryError(props: { message: string; onRetry: () => void; retryLabel: string }) {
  return (
    <div role="alert" className="flex items-center justify-between gap-3 py-4 text-body-xs-regular text-danger-primary">
      <span>{props.message}</span>
      <Button variant="secondary" size="sm" onClick={props.onRetry}>
        {props.retryLabel}
      </Button>
    </div>
  );
}

function WorkspaceRosterReadErrors(props: {
  rosterError: boolean;
  onRetryRoster: () => void;
  membershipError: boolean;
  onRetryMembership: () => void;
  invitationError: boolean;
  onRetryInvitations: () => void;
  message: string;
  retryLabel: string;
}) {
  return (
    <>
      {props.rosterError && (
        <QueryError message={props.message} retryLabel={props.retryLabel} onRetry={props.onRetryRoster} />
      )}
      {props.membershipError && (
        <QueryError message={props.message} retryLabel={props.retryLabel} onRetry={props.onRetryMembership} />
      )}
      {!props.membershipError && props.invitationError && (
        <QueryError message={props.message} retryLabel={props.retryLabel} onRetry={props.onRetryInvitations} />
      )}
    </>
  );
}

function matchesSearch(member: IWorkspaceMember, searchQuery: string) {
  const details = member.member;
  const searchable = `${details.first_name} ${details.last_name} ${details.display_name} ${details.email ?? ""}`;
  return searchable.toLowerCase().includes(searchQuery.toLowerCase());
}

function matchesInvitation(invitation: IWorkspaceMemberInvitation, searchQuery: string) {
  return invitation.email.toLowerCase().includes(searchQuery.toLowerCase());
}

function getVisibleWorkspaceMembers(members: IWorkspaceMember[], searchQuery: string, filters: IMemberFilters) {
  const memberDetailsMap = Object.fromEntries(
    members.map(({ member, created_at }) => [member.id, { ...member, joining_date: created_at }] as const)
  );
  const sortedMembers = sortWorkspaceMembers(
    members.filter((member) => !member.member.is_bot),
    memberDetailsMap,
    (member) => member.member.id,
    filters
  ).filter((member) => !searchQuery || matchesSearch(member, searchQuery));
  return sortBy(sortedMembers, (member) => member.is_active !== true);
}

function getVisibleWorkspaceInvitations(invitations: IWorkspaceMemberInvitation[], searchQuery: string) {
  return invitations.filter((invitation) => !searchQuery || matchesInvitation(invitation, searchQuery));
}

function hasWorkspaceReadError(rosterError: boolean, membershipError: boolean, invitationError: boolean) {
  return rosterError || membershipError || invitationError;
}

export const WorkspaceMembersList = function WorkspaceMembersList(props: {
  searchQuery: string;
  isAdmin: boolean;
  filters: IMemberFilters;
  onFilterUpdate: (updates: Partial<IMemberFilters>) => void;
}) {
  const { searchQuery, isAdmin, filters, onFilterUpdate } = props;
  const [showPendingInvites, setShowPendingInvites] = useState<boolean>(true);

  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const roster = useWorkspaceMemberRoster(workspaceSlug);
  const invitations = useWorkspaceInvitations(workspaceSlug);
  const { t } = useTranslation();

  const hasReadError = hasWorkspaceReadError(
    roster.query.isError,
    invitations.membershipQuery.isError,
    invitations.isError
  );
  const shouldShowLoader = !hasReadError && !roster.members && !invitations.data;
  if (shouldShowLoader) return <MembersSettingsLoader />;

  const searchedMembers = getVisibleWorkspaceMembers(roster.members ?? [], searchQuery, filters);
  const searchedInvitations = getVisibleWorkspaceInvitations(invitations.data ?? [], searchQuery);
  const retryLabel = t("retry");
  const showNoMatches =
    roster.query.isSuccess && invitations.isSuccess && searchedInvitations.length === 0 && searchedMembers.length === 0;
  const showPendingInvitations = isAdmin && searchedInvitations.length > 0;

  return (
    <>
      <WorkspaceRosterReadErrors
        rosterError={roster.query.isError}
        onRetryRoster={() => void roster.query.refetch()}
        membershipError={invitations.membershipQuery.isError}
        onRetryMembership={() => void invitations.membershipQuery.refetch()}
        invitationError={invitations.isError}
        onRetryInvitations={() => void invitations.refetch()}
        message={t("something_went_wrong_please_try_again")}
        retryLabel={retryLabel}
      />
      <div className="divide-y-[0.5px] divide-subtle overflow-scroll">
        {searchedMembers.length !== 0 && (
          <WorkspaceMembersListItem memberDetails={searchedMembers} filters={filters} onFilterUpdate={onFilterUpdate} />
        )}
        {showNoMatches && (
          <h4 className="mt-16 text-center text-body-xs-regular text-placeholder">{t("no_matching_members")}</h4>
        )}
      </div>
      {showPendingInvitations && (
        <Collapsible
          isOpen={showPendingInvites}
          onToggle={() => setShowPendingInvites((prev) => !prev)}
          buttonClassName="w-full"
          className=""
          title={
            <div className="flex w-full items-center justify-between pt-4">
              <div className="flex">
                <h4 className="pt-2 pb-2 text-h5-medium">{t("workspace_settings.settings.members.pending_invites")}</h4>
                <CountChip count={searchedInvitations.length} className="m-auto ml-2 h-5" />
              </div>{" "}
              <ChevronDownIcon className={`h-5 w-5 transition-all ${showPendingInvites ? "rotate-180" : ""}`} />
            </div>
          }
        >
          <Disclosure.Panel>
            <div className="ml-auto items-center gap-1.5 rounded-md bg-surface-1 py-1.5">
              {searchedInvitations.map((invitation) => (
                <WorkspaceInvitationsListItem key={invitation.id} invitation={invitation} />
              ))}
            </div>
          </Disclosure.Panel>
        </Collapsible>
      )}
    </>
  );
};
