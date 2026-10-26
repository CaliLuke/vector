/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery } from "@tanstack/react-query";
import { currentUserOptions } from "@/queries/instance";
import { useNavigate, Outlet } from "react-router";
import { useEffect } from "react";
// hooks

function RootLayout() {
  // router
  const navigate = useNavigate();
  const user = useQuery(currentUserOptions);
  const isUserLoggedIn = user.isPending ? undefined : user.isSuccess;

  useEffect(() => {
    if (isUserLoggedIn === true) void navigate("/general", { replace: true });
  }, [navigate, isUserLoggedIn]);

  return (
    <div className="relative z-10 flex h-screen w-screen flex-col items-center overflow-hidden overflow-y-auto bg-surface-1 px-8 pt-6 pb-10">
      <Outlet />
    </div>
  );
}

export default RootLayout;
