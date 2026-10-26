/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute, Outlet, useParams } from "@tanstack/react-router";
import { ProjectListFiltersProvider } from "@/hooks/use-project-list-filters";

const meta = () => [
  { name: "robots", content: "noindex, nofollow" },
  { name: "viewport", content: "width=device-width, initial-scale=1, minimum-scale=1, viewport-fit=cover" },
];

function AppLayout(): React.JSX.Element | null {
  const { workspaceSlug } = useParams({ strict: false });
  return (
    <ProjectListFiltersProvider workspaceSlug={workspaceSlug ?? ""}>
      <Outlet />
    </ProjectListFiltersProvider>
  );
}

export const Route = createFileRoute("/_authenticated")({
  component: AppLayout,
  head: () => ({ meta: meta() }),
});
