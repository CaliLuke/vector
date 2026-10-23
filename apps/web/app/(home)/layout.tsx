/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { createFileRoute, Outlet } from "@tanstack/react-router";

// types

const meta = () => [
  { name: "robots", content: "index, nofollow" },
  { name: "viewport", content: "width=device-width, initial-scale=1, minimum-scale=1, viewport-fit=cover" },
];

function HomeLayout(): React.JSX.Element | null {
  return <Outlet />;
}

export const Route = createFileRoute("/_home")({
  component: HomeLayout,
  head: () => ({ meta: meta() }),
});
