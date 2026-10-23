/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute, Outlet } from "@tanstack/react-router";
import { observer } from "mobx-react";

function AutomationsListLayout(): React.JSX.Element | null {
  return <Outlet />;
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/settings/projects/$projectId/automations")({
  component: observer(AutomationsListLayout),
});
