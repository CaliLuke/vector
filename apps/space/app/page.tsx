/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery } from "@tanstack/react-query";
import { currentUserOptions } from "@/queries/session";
import { useEffect } from "react";
import { useSearchParams, useNavigate } from "react-router";
// plane imports
import { isValidNextPath } from "@plane/utils/url";
// components
import { UserLoggedIn } from "@/components/account/user-logged-in";
import { LogoSpinner } from "@/components/common/logo-spinner";
import { AuthView } from "@/components/views";
// hooks
import type { Route } from "./+types/page";

export const headers: Route.HeadersFunction = () => ({
  "X-Frame-Options": "SAMEORIGIN",
});

const HomePage = function HomePage() {
  const { data: currentUser, isSuccess: isAuthenticated, isPending: isInitializing } = useQuery(currentUserOptions);
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const nextPath = searchParams.get("next_path");

  useEffect(() => {
    if (currentUser && isAuthenticated && nextPath && isValidNextPath(nextPath)) {
      void navigate(nextPath, { replace: true });
    }
  }, [currentUser, isAuthenticated, nextPath, navigate]);

  if (isInitializing)
    return (
      <div className="flex h-screen min-h-[500px] w-full items-center justify-center bg-surface-1">
        <LogoSpinner />
      </div>
    );

  if (currentUser && isAuthenticated) {
    if (nextPath && isValidNextPath(nextPath)) {
      return (
        <div className="flex h-screen min-h-[500px] w-full items-center justify-center bg-surface-1">
          <LogoSpinner />
        </div>
      );
    }
    return <UserLoggedIn />;
  }

  return <AuthView />;
};

export default HomePage;
