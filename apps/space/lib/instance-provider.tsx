/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery } from "@tanstack/react-query";
import { currentUserOptions, instanceInfoOptions } from "@/queries/session";
import { Link } from "react-router";
import { useTheme } from "next-themes";
// plane imports
import { SPACE_BASE_PATH } from "@plane/constants";
import { PlaneLockup } from "@plane/propel/icons";
// assets
import PlaneBackgroundPatternDark from "@/app/assets/auth/background-pattern-dark.svg?url";
import PlaneBackgroundPattern from "@/app/assets/auth/background-pattern.svg?url";
// components
import { LogoSpinner } from "@/components/common/logo-spinner";
import { InstanceFailureView } from "@/components/instance/instance-failure-view";
// hooks

export const InstanceProvider = function InstanceProvider({ children }: { children: React.ReactNode }) {
  const { data: instanceInfo, error } = useQuery(instanceInfoOptions);
  const instance = instanceInfo?.instance;
  const { resolvedTheme } = useTheme();

  const patternBackground = resolvedTheme === "dark" ? PlaneBackgroundPatternDark : PlaneBackgroundPattern;

  useQuery(currentUserOptions);

  if (!instance && !error)
    return (
      <div className="flex h-screen w-full items-center justify-center">
        <LogoSpinner />
      </div>
    );

  if (error) {
    return (
      <div className="relative">
        <div className="flex h-screen w-full flex-col overflow-hidden overflow-y-auto">
          <div className="z-50 container mx-auto flex h-[110px] flex-shrink-0 items-center justify-between gap-5 px-5 lg:px-0">
            <div className="flex items-center gap-x-2 py-10">
              <Link to={`${SPACE_BASE_PATH}/`}>
                <PlaneLockup className="h-7 w-auto text-primary" />
              </Link>
            </div>
          </div>
          <div className="absolute inset-0 z-0">
            <img src={patternBackground} className="h-full w-screen object-cover" alt="Vector background pattern" />
          </div>
          <div className="relative z-10 flex-grow">
            <div className="relative mx-auto flex h-full w-full items-center justify-center overflow-y-auto px-6 py-10">
              <InstanceFailureView />
            </div>
          </div>
        </div>
      </div>
    );
  }

  return children;
};
