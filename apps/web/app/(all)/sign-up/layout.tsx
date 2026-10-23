/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { createFileRoute, Outlet } from "@tanstack/react-router";

const meta = () => [{ title: "Sign up - Vector" }, { name: "robots", content: "index, nofollow" }];

function SignUpLayout(): React.JSX.Element | null {
  return <Outlet />;
}

export const Route = createFileRoute("/sign-up")({
  component: SignUpLayout,
  head: () => ({ meta: meta() }),
});
