/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState, useEffect } from "react";

import { LogOut, Settings, Settings2 } from "lucide-react";
// plane imports
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { Avatar, CustomMenu } from "@plane/ui";
import { getFileURL } from "@plane/utils/file";
// components
import { CoverImage } from "@/components/common/cover-image";
import { AppSidebarItem } from "@/components/sidebar/sidebar-item";
// hooks
import { useSidebarActions } from "@/providers/sidebar-state";
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useCurrentUser, useSignOut } from "@/hooks/use-current-user";

export function UserMenuRoot() {
  // states
  const [isUserMenuOpen, setIsUserMenuOpen] = useState(false);
  // store hooks
  const { toggleAnySidebarDropdown } = useSidebarActions();
  const { data: currentUser } = useCurrentUser();
  const { mutateAsync: signOut } = useSignOut();
  const { toggleProfileSettingsModal } = useCommandPalette();
  // translation
  const { t } = useTranslation();

  const handleSignOut = () => {
    signOut().catch(() =>
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("auth.sign_out.toast.error.title"),
        message: t("auth.sign_out.toast.error.message"),
      })
    );
  };

  // Toggle sidebar dropdown state when menu is open
  useEffect(() => {
    if (isUserMenuOpen) toggleAnySidebarDropdown(true);
    else toggleAnySidebarDropdown(false);
  }, [isUserMenuOpen, toggleAnySidebarDropdown]);

  return (
    <CustomMenu
      className="flex items-center"
      ariaLabel={t("aria_labels.projects_sidebar.open_user_menu")}
      customButton={
        <AppSidebarItem
          variant="content"
          item={{
            icon: (
              <Avatar
                name={currentUser?.display_name}
                src={getFileURL(currentUser?.avatar_url ?? "")}
                size={20}
                shape="circle"
              />
            ),
            isActive: isUserMenuOpen,
          }}
        />
      }
      customButtonClassName="cursor-pointer appearance-none border-0 bg-transparent p-0 text-inherit outline-none"
      menuButtonOnClick={() => !isUserMenuOpen && setIsUserMenuOpen(true)}
      onMenuClose={() => setIsUserMenuOpen(false)}
      placement="bottom-end"
      maxHeight="2xl"
      optionsClassName="w-72 p-3 flex flex-col gap-y-3"
      closeOnSelect
    >
      <div className="relative h-29 w-full rounded-lg">
        <CoverImage
          src={currentUser?.cover_image_url ?? undefined}
          alt={currentUser?.display_name}
          className="h-29 w-full rounded-lg"
          showDefaultWhenEmpty
        />
        <div className="absolute inset-0 bg-layer-1/50" />
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2">
          <div className="flex flex-col items-center gap-y-2">
            <div>
              <Avatar
                name={currentUser?.display_name}
                src={getFileURL(currentUser?.avatar_url ?? "")}
                size={40}
                shape="circle"
                className="text-18 font-medium"
              />
            </div>
            <div className="text-center">
              <p className="text-body-sm-medium">
                {currentUser?.first_name} {currentUser?.last_name}
              </p>
              <p className="text-caption-md-regular">{currentUser?.email}</p>
            </div>
          </div>
        </div>
      </div>
      <div>
        <CustomMenu.MenuItem
          onClick={() =>
            toggleProfileSettingsModal({
              activeTab: "general",
              isOpen: true,
            })
          }
          className="flex items-center gap-2"
        >
          <Settings className="size-3.5 shrink-0" />
          {t("settings")}
        </CustomMenu.MenuItem>
        <CustomMenu.MenuItem
          onClick={() =>
            toggleProfileSettingsModal({
              activeTab: "preferences",
              isOpen: true,
            })
          }
          className="flex items-center gap-2"
        >
          <Settings2 className="size-3.5 shrink-0" />
          {t("preferences")}
        </CustomMenu.MenuItem>
      </div>
      <CustomMenu.MenuItem onClick={handleSignOut} className="flex items-center gap-2">
        <LogOut className="size-3.5 shrink-0" />
        {t("sign_out")}
      </CustomMenu.MenuItem>
    </CustomMenu>
  );
}
