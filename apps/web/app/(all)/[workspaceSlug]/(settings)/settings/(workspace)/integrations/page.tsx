/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
import { skipToken } from "@tanstack/react-query";
// components
import { EUserPermissions } from "@plane/constants";
import { NotAuthorizedView } from "@/components/auth-screens/not-authorized-view";
import { PageHead } from "@/components/core/page-title";
import { SingleIntegrationCard } from "@/components/integration/single-integration-card";
import { IntegrationAndImportExportBanner } from "@/components/ui/integration-and-import-export-banner";
import { IntegrationsSettingsLoader } from "@/components/ui/loader/settings/integration";
// constants
import { APP_INTEGRATIONS } from "@plane/constants";
// hooks
import { useCurrentWorkspace } from "@/hooks/use-workspaces";
import { useWorkspacePermissions } from "@/hooks/use-permissions";
// services
import { IntegrationService } from "@/services/integrations";

const integrationService = new IntegrationService();

function WorkspaceIntegrationsPage() {
  // store hooks
  const { data: currentWorkspace } = useCurrentWorkspace();
  const { can: canWorkspace, isPending, isError } = useWorkspacePermissions(currentWorkspace?.slug);

  // derived values
  const isAdmin = canWorkspace([EUserPermissions.ADMIN]);
  const pageTitle = currentWorkspace?.name ? `${currentWorkspace.name} - Integrations` : undefined;
  const { data: appIntegrations } = useRecordCollection({
    queryKey: APP_INTEGRATIONS,
    queryFn: isAdmin ? () => integrationService.getAppIntegrationsList() : skipToken,
  });

  if (isPending) return <IntegrationsSettingsLoader />;

  if (isError || !isAdmin) return <NotAuthorizedView section="settings" className="h-auto" />;

  return (
    <>
      <PageHead title={pageTitle} />
      <section className="w-full overflow-y-auto">
        <IntegrationAndImportExportBanner bannerName="Integrations" />
        <div>
          {appIntegrations ? (
            appIntegrations.map((integration) => (
              <SingleIntegrationCard key={integration.id} integration={integration} />
            ))
          ) : (
            <IntegrationsSettingsLoader />
          )}
        </div>
      </section>
    </>
  );
}

export default WorkspaceIntegrationsPage;
