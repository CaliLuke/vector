/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute, useParams, Outlet, useRouter } from "@tanstack/react-router";
import { useEffect } from "react";

// hooks
import { useJoinedProjectIds } from "@/hooks/use-joined-project-ids";

// types

function ProjectSettingsLayout(): React.JSX.Element | null {
  const params = useParams({ strict: false });
  const { workspaceSlug, projectId } = params;
  // router
  const router = useRouter();
  // store hooks
  const joinedProjectIds = useJoinedProjectIds(typeof workspaceSlug === "string" ? workspaceSlug : undefined);

  useEffect(() => {
    if (projectId) return;
    if (joinedProjectIds.length > 0) {
      void router.navigate({ href: `/${workspaceSlug}/settings/projects/${joinedProjectIds[0]}` });
    }
  }, [joinedProjectIds, router, workspaceSlug, projectId]);

  return <Outlet />;
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/settings/projects")({
  component: ProjectSettingsLayout,
});
