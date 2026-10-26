/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute } from "@tanstack/react-router";
// component
import { EUserPermissions } from "@plane/constants";
import { NotAuthorizedView } from "@/components/auth-screens/not-authorized-view";
import { PageHead } from "@/components/core/page-title";
import { SettingsContentWrapper } from "@/components/settings/content-wrapper";
// hooks
import { useCurrentWorkspace } from "@/hooks/use-workspaces";
import { useWorkspacePermissions } from "@/hooks/use-permissions";
// local imports
import { BillingWorkspaceSettingsHeader } from "./header";
import { BillingRoot } from "@/components/workspace/billing";

function BillingSettingsPage(): React.JSX.Element | null {
  const { workspaceSlug } = Route.useParams();
  // store hooks
  const { data: currentWorkspace } = useCurrentWorkspace();
  const { can: canWorkspace, isPending, isError } = useWorkspacePermissions(workspaceSlug);
  // derived values
  const canPerformWorkspaceAdminActions = canWorkspace([EUserPermissions.ADMIN]);
  const pageTitle = currentWorkspace?.name ? `${currentWorkspace.name} - Billing & Plans` : undefined;

  if (isPending) return null;

  if (isError || !canPerformWorkspaceAdminActions) {
    return <NotAuthorizedView section="settings" className="h-auto" />;
  }

  return (
    <SettingsContentWrapper header={<BillingWorkspaceSettingsHeader />} hugging>
      <PageHead title={pageTitle} />
      <BillingRoot />
    </SettingsContentWrapper>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/settings/_workspace/billing")({
  component: BillingSettingsPage,
});
