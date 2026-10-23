/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { createFileRoute, Outlet } from "@tanstack/react-router";

function SetPasswordLayout(): React.JSX.Element | null {
  return <Outlet />;
}

const meta = () => [{ title: "Set Password - Vector" }];

export const Route = createFileRoute("/accounts/set-password")({
  component: SetPasswordLayout,
  head: () => ({ meta: meta() }),
});
