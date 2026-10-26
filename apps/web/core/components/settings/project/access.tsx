/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { EUserPermissions } from "@plane/types";
import { NotAuthorizedView } from "@/components/auth-screens/not-authorized-view";
import { useProjectPermissions } from "@/hooks/use-permissions";

export function ProjectSettingsAccess({
  workspaceSlug,
  projectId,
  children,
}: {
  workspaceSlug: string;
  projectId: string;
  children: ReactNode;
}) {
  const { can, isPending } = useProjectPermissions(workspaceSlug, projectId);
  if (isPending) return null;
  if (!can([EUserPermissions.ADMIN])) return <NotAuthorizedView section="settings" isProjectView className="h-auto" />;
  return children;
}
