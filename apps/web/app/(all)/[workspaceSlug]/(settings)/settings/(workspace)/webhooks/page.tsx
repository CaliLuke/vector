/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file in the project root for details.
 */

import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { EmptyStateCompact } from "@plane/propel/empty-state";
import { NotAuthorizedView } from "@/components/auth-screens/not-authorized-view";
import { PageHead } from "@/components/core/page-title";
import { SettingsHeading } from "@/components/settings/heading";
import { WebhookSettingsLoader } from "@/components/ui/loader/settings/web-hook";
import { SettingsContentWrapper } from "@/components/settings/content-wrapper";
import { WebhooksList, CreateWebhookModal } from "@/components/web-hooks";
import { webhooksListOptions } from "@/queries/webhooks";
import { useCurrentWorkspace } from "@/hooks/use-workspaces";
import { useWorkspacePermissions } from "@/hooks/use-permissions";
import { WebhooksWorkspaceSettingsHeader } from "./header";

function WebhooksListPage(): React.JSX.Element | null {
  const [showCreateWebhookModal, setShowCreateWebhookModal] = useState(false);
  const { workspaceSlug } = Route.useParams();
  const { t } = useTranslation();
  const { data: currentWorkspace } = useCurrentWorkspace();
  const { can: canWorkspace, isPending, isError } = useWorkspacePermissions(workspaceSlug);
  const canPerformWorkspaceAdminActions = canWorkspace([EUserPermissions.ADMIN]);
  const webhooksQuery = useQuery(webhooksListOptions(workspaceSlug, canPerformWorkspaceAdminActions));

  const pageTitle = currentWorkspace?.name
    ? `${currentWorkspace.name} - ${t("workspace_settings.settings.webhooks.title")}`
    : undefined;

  if (isPending) return <WebhookSettingsLoader />;

  if (isError || !canPerformWorkspaceAdminActions) {
    return <NotAuthorizedView section="settings" className="h-auto" />;
  }

  if (webhooksQuery.isPending) return <WebhookSettingsLoader />;

  return (
    <SettingsContentWrapper header={<WebhooksWorkspaceSettingsHeader />}>
      <PageHead title={pageTitle} />
      <div className="w-full">
        <CreateWebhookModal
          key={`${workspaceSlug}/${showCreateWebhookModal}`}
          currentWorkspace={currentWorkspace ?? null}
          isOpen={showCreateWebhookModal}
          onClose={() => setShowCreateWebhookModal(false)}
        />
        <SettingsHeading
          title={t("workspace_settings.settings.webhooks.title")}
          description={t("workspace_settings.settings.webhooks.description")}
          control={
            <Button variant="primary" size="lg" onClick={() => setShowCreateWebhookModal(true)}>
              {t("workspace_settings.settings.webhooks.add_webhook")}
            </Button>
          }
        />
        {webhooksQuery.isError ? (
          <div className="mt-6 space-y-3" role="alert">
            <p className="text-13 text-danger-primary">Webhooks could not be loaded. Please try again.</p>
            <Button variant="secondary" onClick={() => void webhooksQuery.refetch()}>
              Retry
            </Button>
          </div>
        ) : webhooksQuery.data?.length ? (
          <div className="mt-4">
            <WebhooksList workspaceSlug={workspaceSlug} webhooks={webhooksQuery.data} />
          </div>
        ) : (
          <div className="flex h-full w-full flex-col">
            <div className="flex h-full w-full items-center justify-center">
              <EmptyStateCompact
                assetKey="webhook"
                title={t("settings_empty_state.webhooks.title")}
                description={t("settings_empty_state.webhooks.description")}
                actions={[
                  {
                    label: t("settings_empty_state.webhooks.cta_primary"),
                    onClick: () => setShowCreateWebhookModal(true),
                  },
                ]}
                align="start"
                rootClassName="py-20"
              />
            </div>
          </div>
        )}
      </div>
    </SettingsContentWrapper>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/settings/_workspace/webhooks")({
  component: WebhooksListPage,
});
