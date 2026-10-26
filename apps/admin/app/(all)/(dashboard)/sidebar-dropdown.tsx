/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { currentUserOptions } from "@/queries/instance";
import { useSidebar } from "@/providers/sidebar";
import { Fragment, useRef } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { setToast, TOAST_TYPE } from "@plane/propel/toast";
import { useTheme as useNextTheme } from "next-themes";
import { LogOut, UserCog2, Palette } from "lucide-react";
import { Menu, Transition } from "@headlessui/react";
// plane internal packages
import { API_BASE_URL } from "@plane/constants";
import { AuthService } from "@plane/services";
import { Avatar } from "@plane/ui";
import { getFileURL } from "@plane/utils/file";
import { cn } from "@plane/utils/common";
// hooks

// service initialization
const authService = new AuthService();

export const AdminSidebarDropdown = function AdminSidebarDropdown() {
  const { isSidebarCollapsed } = useSidebar();
  const { data: currentUser } = useQuery(currentUserOptions);
  const queryClient = useQueryClient();
  // hooks
  const { resolvedTheme, setTheme } = useNextTheme();
  // state
  const formElement = useRef<HTMLFormElement>(null);
  const form = useForm({
    defaultValues: {},
    onSubmit: async () => {
      try {
        await authService.submitForm(formElement.current);
        queryClient.clear();
        setTheme("system");
      } catch (error) {
        setToast({ type: TOAST_TYPE.ERROR, title: "Unable to sign out", message: "Please try again." });
        throw error;
      }
    },
  });
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);

  const handleThemeSwitch = () => {
    const newTheme = resolvedTheme === "dark" ? "light" : "dark";
    setTheme(newTheme);
  };

  const getSidebarMenuItems = () => (
    <Menu.Items
      className={cn(
        "shadow-lg absolute left-0 z-20 mt-1.5 flex w-52 flex-col divide-y divide-subtle rounded-md border border-subtle bg-surface-1 px-1 py-2 text-11 outline-none",
        {
          "left-4": isSidebarCollapsed,
        }
      )}
    >
      <div className="flex flex-col gap-2.5 pb-2">
        <span className="truncate px-2 text-secondary">{currentUser?.email}</span>
      </div>
      <div className="py-2">
        <Menu.Item
          as="button"
          type="button"
          className="flex w-full items-center gap-2 rounded-sm px-2 py-1 hover:bg-layer-1-hover"
          onClick={handleThemeSwitch}
        >
          <Palette className="h-4 w-4 stroke-[1.5]" />
          Switch to {resolvedTheme === "dark" ? "light" : "dark"} mode
        </Menu.Item>
      </div>
      <div className="py-2">
        <form
          method="POST"
          action={`${API_BASE_URL}/api/instances/admins/sign-out/`}
          ref={formElement}
          onSubmit={(event) => {
            event.preventDefault();
            void form.handleSubmit().catch(() => {
              // The sign-out failure is already reported with a toast.
            });
          }}
        >
          <input type="hidden" name="csrfmiddlewaretoken" defaultValue="" />
          <Menu.Item
            as="button"
            type="button"
            disabled={isSubmitting}
            onClick={(event) => {
              event.preventDefault();
              void form.handleSubmit().catch(() => {
                // The sign-out failure is already reported with a toast.
              });
            }}
            className="flex w-full items-center gap-2 rounded-sm px-2 py-1 hover:bg-layer-1-hover"
          >
            <LogOut className="h-4 w-4 stroke-[1.5]" />
            Sign out
          </Menu.Item>
        </form>
      </div>
    </Menu.Items>
  );

  return (
    <div className="flex max-h-header items-center gap-x-5 gap-y-2 border-b border-subtle px-4 py-2.5">
      <div className="h-full w-full truncate">
        <div
          className={`flex flex-grow items-center gap-x-2 truncate rounded-sm ${
            isSidebarCollapsed ? "justify-center" : ""
          }`}
        >
          <Menu as="div" className="flex-shrink-0">
            <Menu.Button
              className={cn("grid place-items-center outline-none", {
                "cursor-default": !isSidebarCollapsed,
              })}
            >
              <div className="flex size-8 flex-shrink-0 items-center justify-center rounded-sm bg-layer-1">
                <UserCog2 className="size-5 text-primary" />
              </div>
            </Menu.Button>
            {isSidebarCollapsed && (
              <Transition
                as={Fragment}
                enter="transition ease-out duration-100"
                enterFrom="transform opacity-0 scale-95"
                enterTo="transform opacity-100 scale-100"
                leave="transition ease-in duration-75"
                leaveFrom="transform opacity-100 scale-100"
                leaveTo="transform opacity-0 scale-95"
              >
                {getSidebarMenuItems()}
              </Transition>
            )}
          </Menu>

          {!isSidebarCollapsed && (
            <div className="flex w-full gap-2">
              <h4 className="grow truncate text-body-md-medium text-primary">Instance admin</h4>
            </div>
          )}
        </div>
      </div>

      {!isSidebarCollapsed && currentUser && (
        <Menu as="div" className="relative flex-shrink-0">
          <Menu.Button className="grid place-items-center outline-none">
            <Avatar
              name={currentUser.display_name}
              src={getFileURL(currentUser.avatar_url)}
              size={24}
              shape="square"
              className="!text-body-sm-medium"
            />
          </Menu.Button>

          <Transition
            as={Fragment}
            enter="transition ease-out duration-100"
            enterFrom="transform opacity-0 scale-95"
            enterTo="transform opacity-100 scale-100"
            leave="transition ease-in duration-75"
            leaveFrom="transform opacity-100 scale-100"
            leaveTo="transform opacity-0 scale-95"
          >
            {getSidebarMenuItems()}
          </Transition>
        </Menu>
      )}
    </div>
  );
};
