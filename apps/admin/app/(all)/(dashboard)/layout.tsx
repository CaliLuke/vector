/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery } from "@tanstack/react-query";
import { currentUserOptions } from "@/queries/instance";
import { useNavigate, Outlet } from "react-router";
import { useEffect } from "react";
// components
import { AdminHeader } from "@/components/common/header";
import { LogoSpinner } from "@/components/common/logo-spinner";
import { NewUserPopup } from "@/components/common/new-user-popup";
// hooks
// local components
import type { Route } from "./+types/layout";
import { AdminSidebar } from "./sidebar";

function AdminLayout(_props: Route.ComponentProps) {
  // router
  const navigate = useNavigate();
  const user = useQuery(currentUserOptions);
  const isUserLoggedIn = user.isPending ? undefined : user.isSuccess;

  useEffect(() => {
    if (isUserLoggedIn === false) void navigate("/", { replace: true });
  }, [navigate, isUserLoggedIn]);

  if (isUserLoggedIn === undefined) {
    return (
      <div className="relative flex h-screen w-full items-center justify-center">
        <LogoSpinner />
      </div>
    );
  }

  if (isUserLoggedIn) {
    return (
      <div className="relative flex h-screen w-screen overflow-hidden">
        <AdminSidebar />
        <main className="relative flex h-full w-full flex-col overflow-hidden bg-surface-1">
          <AdminHeader />
          <div className="vertical-scrollbar scrollbar-md h-full w-full overflow-hidden overflow-y-scroll">
            <Outlet />
          </div>
        </main>
        <NewUserPopup />
      </div>
    );
  }

  return <></>;
}

export default AdminLayout;
