/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link, useParams } from "@tanstack/react-router";

import { Settings, UserPlus } from "lucide-react";
import { Menu } from "@headlessui/react";
// plane imports
import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { CheckIcon } from "@plane/propel/icons";
import type { IWorkspace } from "@plane/types";
import { cn } from "@plane/utils/common";
import { getFileURL } from "@plane/utils/file";
import { getUserRole } from "@plane/utils/permission";
import { useWorkspacePermissions } from "@/hooks/use-permissions";

type TProps = {
  workspace: IWorkspace;
  activeWorkspace: IWorkspace | null;
  handleItemClick: () => void;
  handleWorkspaceNavigation: (workspace: IWorkspace) => void;
  handleClose: () => void;
};
function SidebarDropdownItem(props: TProps) {
  const { workspace, activeWorkspace, handleItemClick, handleWorkspaceNavigation, handleClose } = props;
  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // hooks
  const { t } = useTranslation();
  const { can } = useWorkspacePermissions(activeWorkspace?.slug);

  return (
    <div
      className={cn("px-4 py-2", {
        "bg-layer-transparent-active": workspace.id === activeWorkspace?.id,
        "hover:bg-layer-transparent-hover": workspace.id !== activeWorkspace?.id,
      })}
    >
      <Menu.Item>
        <Link
          to="/$workspaceSlug/"
          params={{ workspaceSlug: workspace.slug }}
          onClick={() => {
            handleWorkspaceNavigation(workspace);
            handleItemClick();
          }}
          className="block w-full"
          id={workspace.id}
        >
          <div className="flex items-center justify-between gap-1 rounded-sm p-1 text-13 text-primary">
            <div className="relative flex w-[80%] items-center justify-start gap-2.5">
              <span
                className={`relative flex h-8 w-8 flex-shrink-0 items-center justify-center border-subtle p-2 text-14 font-medium uppercase ${
                  !workspace?.logo_url && "rounded-md bg-[#026292] text-on-color"
                }`}
              >
                {workspace?.logo_url && workspace.logo_url !== "" ? (
                  <img
                    src={getFileURL(workspace.logo_url)}
                    className="absolute top-0 left-0 h-full w-full rounded-sm object-cover"
                    alt={t("workspace_logo")}
                  />
                ) : (
                  (workspace?.name?.[0] ?? "...")
                )}
              </span>
              <div className="w-[inherit]">
                <div
                  className={`truncate text-left text-13 font-medium text-ellipsis ${workspaceSlug === workspace.slug ? "" : "text-secondary"}`}
                >
                  {workspace.name}
                </div>
                <div className="flex w-fit gap-2 text-13 text-tertiary capitalize">
                  <span>{getUserRole(workspace.role)?.toLowerCase() || "guest"}</span>
                  <div className="m-auto h-1 w-1 rounded-full bg-layer-1/50" />
                  <span className="capitalize">{t("member", { count: workspace.total_members || 0 })}</span>
                </div>
              </div>
            </div>
            {workspace.id === activeWorkspace?.id ? (
              <span className="flex-shrink-0 p-1">
                <CheckIcon className="h-5 w-5 text-primary" />
              </span>
            ) : null}
          </div>
        </Link>
      </Menu.Item>
      {workspace.id === activeWorkspace?.id && (
        <div className="mt-2 mb-1 flex gap-2">
          {can([EUserPermissions.ADMIN, EUserPermissions.MEMBER]) && (
            <Menu.Item>
              <Link
                to="/$workspaceSlug/settings"
                params={{ workspaceSlug: workspace.slug }}
                onClick={(e) => {
                  e.stopPropagation();
                  handleClose();
                }}
                className="flex gap-1.5 rounded-md border border-strong bg-layer-2 px-2.5 py-1.5 text-secondary transition-colors hover:border-strong hover:text-secondary hover:shadow-raised-100"
              >
                <Settings className="my-auto h-4 w-4 flex-shrink-0" />
                <span className="my-auto text-13 font-medium whitespace-nowrap">{t("settings")}</span>
              </Link>
            </Menu.Item>
          )}
          {can([EUserPermissions.ADMIN]) && (
            <Menu.Item>
              <Link
                to="/$workspaceSlug/settings/members"
                params={{ workspaceSlug: workspace.slug }}
                onClick={(e) => {
                  e.stopPropagation();
                  handleClose();
                }}
                className="flex gap-1.5 rounded-md border border-strong bg-layer-2 px-2.5 py-1.5 text-secondary transition-colors hover:border-strong hover:text-secondary hover:shadow-raised-100"
              >
                <UserPlus className="my-auto h-4 w-4 flex-shrink-0" />
                <span className="my-auto text-13 font-medium whitespace-nowrap">
                  {t("project_settings.members.invite_members.title")}
                </span>
              </Link>
            </Menu.Item>
          )}
        </div>
      )}
    </div>
  );
}

export default SidebarDropdownItem;
