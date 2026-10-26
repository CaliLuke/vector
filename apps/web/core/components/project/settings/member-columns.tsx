/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link } from "@tanstack/react-router";

import { CircleMinus } from "lucide-react";
import { Disclosure } from "@headlessui/react";
// plane imports
import { ROLE, EUserPermissions, MEMBER_TRACKER_ELEMENTS } from "@plane/constants";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { EUserProjectRoles, EUserWorkspaceRoles } from "@plane/types";
import type { IWorkspaceMember, IUser, IUserLite, TProjectMembership, TUserPermissions } from "@plane/types";
import { CustomMenu, CustomSelect } from "@plane/ui";
import { getFileURL } from "@plane/utils/file";
// hooks
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useUpdateProjectMemberRole } from "@/hooks/use-member-roster-mutations";

export interface RowData extends Pick<TProjectMembership, "original_role"> {
  member: IUserLite;
  workspaceRole: IWorkspaceMember["role"];
}

type NameProps = {
  rowData: RowData;
  workspaceSlug: string;
  isAdmin: boolean;
  currentUser: IUser | undefined;
  setRemoveMemberModal: (rowData: RowData) => void;
};

type AccountTypeProps = {
  rowData: RowData;
  workspaceSlug: string;
  projectId: string;
};

const PROJECT_ROLE_PERMISSIONS: Record<EUserProjectRoles, TUserPermissions> = {
  [EUserProjectRoles.ADMIN]: EUserPermissions.ADMIN,
  [EUserProjectRoles.MEMBER]: EUserPermissions.MEMBER,
  [EUserProjectRoles.GUEST]: EUserPermissions.GUEST,
};

export function NameColumn(props: NameProps) {
  const { rowData, workspaceSlug, isAdmin, currentUser, setRemoveMemberModal } = props;
  // derived values
  const { avatar_url, display_name, email, first_name, id, last_name } = rowData.member;

  return (
    <Disclosure>
      {() => (
        <div className="group relative">
          <div className="flex w-72 items-center gap-2">
            <div className="flex flex-1 items-center gap-x-2 gap-y-2">
              {avatar_url && avatar_url.trim() !== "" ? (
                <Link to="/$workspaceSlug/profile/$userId/" params={{ workspaceSlug: workspaceSlug, userId: id }}>
                  <span className="relative flex size-6 items-center justify-center rounded-full text-on-color capitalize">
                    <img
                      src={getFileURL(avatar_url)}
                      className="absolute top-0 left-0 h-full w-full rounded-full object-cover"
                      alt={display_name || email}
                    />
                  </span>
                </Link>
              ) : (
                <Link to="/$workspaceSlug/profile/$userId/" params={{ workspaceSlug: workspaceSlug, userId: id }}>
                  <span className="relative flex size-6 items-center justify-center rounded-full bg-layer-3 text-11 text-on-color capitalize">
                    {(email ?? display_name ?? "?")[0]}
                  </span>
                </Link>
              )}
              {first_name} {last_name}
            </div>
            {(isAdmin || id === currentUser?.id) && (
              <CustomMenu
                ellipsis
                buttonClassName="p-0.5 opacity-0 group-hover:opacity-100 transition-opacity"
                optionsClassName="p-1.5"
                placement="bottom-end"
              >
                <CustomMenu.MenuItem
                  onClick={() => setRemoveMemberModal(rowData)}
                  className="flex cursor-pointer items-center gap-x-1 font-medium text-danger-primary"
                >
                  <span data-ph-element={MEMBER_TRACKER_ELEMENTS.PROJECT_MEMBER_TABLE_CONTEXT_MENU}>
                    <CircleMinus className="size-3.5 flex-shrink-0" />
                    {rowData.member?.id === currentUser?.id ? "Leave " : "Remove "}
                  </span>
                </CustomMenu.MenuItem>
              </CustomMenu>
            )}
          </div>
        </div>
      )}
    </Disclosure>
  );
}

export function AccountTypeColumn(props: AccountTypeProps) {
  const { rowData, projectId, workspaceSlug } = props;
  const { data: currentUser } = useCurrentUser();
  const { projectRole: currentProjectRole, workspaceRole } = useProjectPermissions(workspaceSlug, projectId);
  const roleMutation = useUpdateProjectMemberRole(workspaceSlug, projectId, rowData.member.id, currentProjectRole);
  // derived values
  const roleLabel = ROLE[rowData.original_role ?? EUserPermissions.GUEST];
  const isCurrentUser = currentUser?.id === rowData.member.id;
  const isRowDataWorkspaceAdmin = rowData.workspaceRole === EUserWorkspaceRoles.ADMIN;
  const isRowDataWorkspaceGuest = rowData.workspaceRole === EUserWorkspaceRoles.GUEST;
  const isCurrentUserWorkspaceAdmin = workspaceRole === EUserWorkspaceRoles.ADMIN;
  const isCurrentUserProjectAdmin = currentProjectRole === EUserPermissions.ADMIN;
  const currentProjectRoleValue =
    currentProjectRole === undefined ? undefined : Number.parseInt(String(currentProjectRole), 10);
  const memberProjectRoleValue = Number.parseInt(String(rowData.original_role ?? EUserPermissions.GUEST), 10);

  // logic
  // Workspace admin can change his own role
  // Project admin can change any role except his own and workspace admin's role
  const isRoleEditable =
    currentProjectRole !== undefined &&
    ((isCurrentUserWorkspaceAdmin && isCurrentUser) ||
      (isCurrentUserProjectAdmin &&
        !isRowDataWorkspaceAdmin &&
        !isCurrentUser &&
        memberProjectRoleValue < (currentProjectRoleValue ?? 0)));
  const allowedRoles = Object.entries(ROLE).filter(([key]) => {
    const role = Number(key);
    if (isRowDataWorkspaceGuest) return key === String(EUserPermissions.GUEST);
    if (isCurrentUserWorkspaceAdmin && isCurrentUser) return true;
    return (
      isCurrentUserProjectAdmin &&
      !isRowDataWorkspaceAdmin &&
      !isCurrentUser &&
      memberProjectRoleValue < (currentProjectRoleValue ?? 0) &&
      role < (currentProjectRoleValue ?? 0)
    );
  });
  return (
    <>
      {isRoleEditable ? (
        <CustomSelect
          value={rowData.original_role}
          onChange={(value: EUserProjectRoles) => {
            void roleMutation
              .mutateAsync(PROJECT_ROLE_PERMISSIONS[value])
              .catch((error: { error?: string | string[] }) => {
                const message = Array.isArray(error?.error) ? error.error[0] : error?.error;
                setToast({
                  type: TOAST_TYPE.ERROR,
                  title: "You can’t change this role yet.",
                  message: message ?? "An error occurred while updating member role. Please try again.",
                });
              });
          }}
          disabled={roleMutation.isPending}
          label={
            <div className="flex">
              <span>{roleLabel}</span>
            </div>
          }
          buttonClassName="!px-0 !justify-start hover:bg-surface-1 border-none"
          className="w-32 rounded-md p-0"
          input
        >
          {allowedRoles.map(([key, label]) => (
            <CustomSelect.Option key={key} value={Number(key)}>
              {label}
            </CustomSelect.Option>
          ))}
        </CustomSelect>
      ) : (
        <div className="flex w-32">
          <span>{roleLabel}</span>
        </div>
      )}
    </>
  );
}
