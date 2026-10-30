/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useState } from "react";

// plane imports
import { ROLE, EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { LinkIcon, TrashIcon, ChevronDownIcon } from "@plane/propel/icons";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { TContextMenuItem } from "@plane/ui";
import { CustomSelect, CustomMenu } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { copyTextToClipboard } from "@plane/utils/string";
// components
import { ConfirmWorkspaceMemberRemove } from "@/components/workspace/confirm-workspace-member-remove";
// hooks
import { useWorkspacePermissions } from "@/hooks/use-permissions";
import { useDeleteWorkspaceInvitation, useUpdateWorkspaceInvitation } from "@/hooks/use-member-roster-mutations";
import type { IWorkspaceMemberInvitation } from "@plane/types";

type Props = {
  invitation: IWorkspaceMemberInvitation;
};

export function WorkspaceInvitationsListItem(props: Props) {
  const { invitation } = props;
  const invitationId = invitation.id;
  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // states
  const [removeMemberModal, setRemoveMemberModal] = useState(false);
  // plane hooks
  const { t } = useTranslation();
  // permissions
  const { workspaceRole: currentWorkspaceRole, can: canWorkspace } = useWorkspacePermissions(workspaceSlug);
  const updateInvitation = useUpdateWorkspaceInvitation(workspaceSlug ?? "", invitationId);
  const deleteInvitation = useDeleteWorkspaceInvitation(workspaceSlug ?? "", invitationId);
  // is the current logged in user admin
  const isAdmin = canWorkspace([EUserPermissions.ADMIN]);
  // role change access-
  // 1. user cannot change their own role
  // 2. only admin or member can change role
  // 3. user cannot change role of higher role
  const hasRoleChangeAccess = canWorkspace([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  const handleRemoveInvitation = async () => {
    if (!workspaceSlug) throw new Error("A workspace is required.");
    await deleteInvitation.mutateAsync();
    setToast({
      type: TOAST_TYPE.SUCCESS,
      title: "Success!",
      message: "Invitation removed successfully.",
    });
  };

  if (!currentWorkspaceRole) return null;

  const handleCopyText = async () => {
    try {
      const inviteLink = new URL(invitation.invite_link, window.location.origin).href;
      await copyTextToClipboard(inviteLink);
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("common.link_copied"),
        message: t("entity.link_copied_to_clipboard", { entity: t("common.invite") }),
      });
    } catch {
      setToast({ type: TOAST_TYPE.ERROR, title: "Could not copy to clipboard", message: "Please try again." });
    }
  };

  const MENU_ITEMS: TContextMenuItem[] = [
    {
      key: "copy-link",
      action: () => void handleCopyText(),
      title: t("common.actions.copy_link"),
      icon: LinkIcon,
      shouldRender: !!invitation.invite_link,
    },
    {
      key: "remove",
      action: () => {
        setRemoveMemberModal(true);
      },
      title: t("common.remove"),
      icon: TrashIcon,
      shouldRender: isAdmin,
      className: "text-danger-primary",
      iconClassName: "text-danger-primary",
    },
  ];

  return (
    <>
      <ConfirmWorkspaceMemberRemove
        isOpen={removeMemberModal}
        onClose={() => setRemoveMemberModal(false)}
        userDetails={{
          id: invitation.id,
          display_name: `${invitation.email}`,
        }}
        onSubmit={handleRemoveInvitation}
      />
      <div className="group flex h-full w-full items-center justify-between px-3 py-4 hover:bg-layer-transparent-hover">
        <div className="flex items-center gap-x-4 gap-y-2">
          <span className="relative flex h-10 w-10 items-center justify-center rounded-sm bg-layer-3 p-4 text-tertiary capitalize">
            {(invitation.email ?? "?")[0]}
          </span>
          <div>
            <h4 className="cursor-default text-body-xs-regular">{invitation.email}</h4>
          </div>
        </div>
        <div className="flex items-center gap-2 text-11">
          <div className="flex items-center justify-center rounded-sm bg-label-yellow-bg-strong/20 px-2.5 py-1 text-center text-caption-sm-medium text-label-yellow-text">
            <p>{t("common.pending")}</p>
          </div>
          <CustomSelect
            customButton={
              <div className="item-center flex gap-1 rounded-sm px-2 py-0.5">
                <span
                  className={`flex items-center rounded-sm text-caption-sm-medium ${
                    hasRoleChangeAccess ? "" : "text-placeholder"
                  }`}
                >
                  {ROLE[invitation.role]}
                </span>
                {hasRoleChangeAccess && (
                  <span className="grid place-items-center">
                    <ChevronDownIcon className="h-3 w-3" />
                  </span>
                )}
              </div>
            }
            value={invitation.role}
            onChange={(value: EUserPermissions) => {
              if (!workspaceSlug || !value) return;

              updateInvitation.mutate(
                { role: value },
                {
                  onError: (err: unknown) => {
                    const error = err as { error?: string };
                    setToast({
                      type: TOAST_TYPE.ERROR,
                      title: "Error!",
                      message: error?.error || "An error occurred while updating member role. Please try again.",
                    });
                  },
                }
              );
            }}
            disabled={!hasRoleChangeAccess || updateInvitation.isPending}
            placement="bottom-end"
          >
            {Object.keys(ROLE).map((key) => {
              if (
                currentWorkspaceRole &&
                Number(currentWorkspaceRole) !== 20 &&
                Number(currentWorkspaceRole) < parseInt(key)
              )
                return null;

              return (
                <CustomSelect.Option key={key} value={parseInt(key, 10)}>
                  <>{ROLE[parseInt(key) as keyof typeof ROLE]}</>
                </CustomSelect.Option>
              );
            })}
          </CustomSelect>
          {isAdmin && (
            <CustomMenu ellipsis placement="bottom-end" closeOnSelect>
              {MENU_ITEMS.map((item) => {
                if (item.shouldRender === false) return null;
                return (
                  <CustomMenu.MenuItem
                    key={item.key}
                    onClick={() => {
                      item.action();
                    }}
                    className={cn(
                      "flex items-center gap-2",
                      {
                        "text-placeholder": item.disabled,
                      },
                      item.className
                    )}
                    disabled={item.disabled}
                  >
                    {item.icon && <item.icon className={cn("h-3 w-3", item.iconClassName)} />}
                    <div>
                      <h5>{item.title}</h5>
                      {item.description && (
                        <p
                          className={cn("whitespace-pre-line text-tertiary", {
                            "text-placeholder": item.disabled,
                          })}
                        >
                          {item.description}
                        </p>
                      )}
                    </div>
                  </CustomMenu.MenuItem>
                );
              })}
            </CustomMenu>
          )}
        </div>
      </div>
    </>
  );
}
