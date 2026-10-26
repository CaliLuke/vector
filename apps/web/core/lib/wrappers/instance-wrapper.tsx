/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { LogoSpinner } from "@/components/common/logo-spinner";
import { InstanceNotReady, MaintenanceView } from "@/components/instance";
import { instanceInfoOptions } from "@/queries/instance";

export default function InstanceWrapper({ children }: { children: ReactNode }) {
  const { data, isPending, isError } = useQuery(instanceInfoOptions);

  if (isPending)
    return (
      <div className="relative flex h-screen w-full items-center justify-center">
        <LogoSpinner />
      </div>
    );

  if (isError) return <MaintenanceView />;
  if (!data.instance.is_setup_done) return <InstanceNotReady />;

  return <>{children}</>;
}
