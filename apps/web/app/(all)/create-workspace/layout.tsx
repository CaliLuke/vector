/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { createFileRoute, Outlet } from "@tanstack/react-router";

function CreateWorkspaceLayout(): React.JSX.Element | null {
  return <Outlet />;
}

const meta = () => [{ title: "Create Workspace" }];

export const Route = createFileRoute("/create-workspace")({
  component: CreateWorkspaceLayout,
  head: () => ({ meta: meta() }),
});
