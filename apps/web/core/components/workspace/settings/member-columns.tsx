/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { isRecord } from "@plane/utils/common";

import { Disclosure } from "@headlessui/react";
// plane imports
import { ROLE, EUserPermissions, MEMBER_TRACKER_ELEMENTS } from "@plane/constants";
import { TrashIcon, SuspendedUserIcon } from "@plane/propel/icons";
import { Pill, EPillVariant, EPillSize } from "@plane/propel/pill";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IUser, IWorkspaceMember } from "@plane/types";
// plane ui
import { CustomSelect, PopoverMenu } from "@plane/ui";
// helpers
import { MemberProfileAvatar } from "@/components/common/member-profile-avatar";
// hooks
import { useWorkspacePermissions } from "@/hooks/use-permissions";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useUpdateWorkspaceMemberRole } from "@/hooks/use-member-roster-mutations";

export type RowData = IWorkspaceMember;

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
};

export function NameColumn(props: NameProps) {
  const { rowData, workspaceSlug, isAdmin, currentUser, setRemoveMemberModal } = props;
  // derived values
  const { first_name, id, last_name } = rowData.member;
  const isSuspended = rowData.is_active === false;

  return (
    <Disclosure>
      {() => (
        <div className="group relative">
          <div className="flex w-72 items-center justify-between gap-x-4 gap-y-2">
            <div className="flex flex-1 items-center gap-x-2 gap-y-2">
              {isSuspended ? (
                <div className="rounded-full bg-layer-1">
                  <SuspendedUserIcon className="size-6 text-placeholder" />
                </div>
              ) : (
                <MemberProfileAvatar
                  member={rowData.member}
                  workspaceSlug={workspaceSlug}
                  fallbackTextClassName="text-tertiary"
                />
              )}
              <span className={isSuspended ? "text-placeholder" : ""}>
                {first_name} {last_name}
              </span>
            </div>

            {!isSuspended && (isAdmin || id === currentUser?.id) && (
              <PopoverMenu
                data={[""]}
                keyExtractor={(item) => item}
                popoverClassName="justify-end"
                buttonClassName="outline-none	origin-center rotate-90 size-8 aspect-square flex-shrink-0 grid place-items-center opacity-0 group-hover:opacity-100 transition-opacity"
                render={() => (
                  <button
                    type="button"
                    className="flex cursor-pointer items-center gap-x-3"
                    onClick={() => setRemoveMemberModal(rowData)}
                    data-ph-element={MEMBER_TRACKER_ELEMENTS.WORKSPACE_MEMBER_TABLE_CONTEXT_MENU}
                  >
                    <TrashIcon className="size-3.5 align-middle" /> {id === currentUser?.id ? "Leave " : "Remove "}
                  </button>
                )}
              />
            )}
          </div>
        </div>
      )}
    </Disclosure>
  );
}

export function AccountTypeColumn(props: AccountTypeProps) {
  const { rowData, workspaceSlug } = props;
  // store hooks
  const { can: canWorkspace } = useWorkspacePermissions(workspaceSlug);

  const { data: currentUser } = useCurrentUser();
  const roleMutation = useUpdateWorkspaceMemberRole(workspaceSlug, rowData.member.id);

  // derived values
  const isCurrentUser = currentUser?.id === rowData.member.id;
  const isAdminRole = canWorkspace([EUserPermissions.ADMIN]);
  const isRoleNonEditable = isCurrentUser || !isAdminRole;
  const isSuspended = rowData.is_active === false;

  return (
    <>
      {isSuspended ? (
        <div className="flex w-32">
          <Pill variant={EPillVariant.DEFAULT} size={EPillSize.SM} className="border-none">
            Suspended
          </Pill>
        </div>
      ) : isRoleNonEditable ? (
        <div className="flex w-32">
          <span>{ROLE[rowData.role]}</span>
        </div>
      ) : (
        <CustomSelect
          value={rowData.role}
          onChange={(value) => {
            if (value === null) return;
            return roleMutation.mutate(value, {
              onError: (error: unknown) => {
                const detail = isRecord(error) ? error.error : undefined;
                const first = Array.isArray(detail) ? detail[0] : detail;
                const message = typeof first === "string" ? first : undefined;
                setToast({
                  type: TOAST_TYPE.ERROR,
                  title: "Error!",
                  message: message ?? "An error occurred while updating member role. Please try again.",
                });
              },
            });
          }}
          disabled={roleMutation.isPending}
          label={
            <div className="flex">
              <span>{ROLE[rowData.role]}</span>
            </div>
          }
          buttonClassName={`!px-0 !justify-start hover:bg-surface-1 ${roleMutation.isError ? "border-danger-strong" : "border-none"}`}
          className="w-32 rounded-md p-0"
          input
        >
          {[EUserPermissions.GUEST, EUserPermissions.MEMBER, EUserPermissions.ADMIN].map((role) => (
            <CustomSelect.Option key={role} value={role}>
              {ROLE[role]}
            </CustomSelect.Option>
          ))}
        </CustomSelect>
      )}
    </>
  );
}
