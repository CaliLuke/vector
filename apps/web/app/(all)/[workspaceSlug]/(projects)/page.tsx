/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute } from "@tanstack/react-router";

// components
import { useTranslation } from "@plane/i18n";
import { AppHeader } from "@/components/core/app-header";
import { ContentWrapper } from "@/components/core/content-wrapper";
import { PageHead } from "@/components/core/page-title";
import { HomeProvider } from "@/components/home/provider";
import { WorkspaceHomeView } from "@/components/home";
// hooks
import { useCurrentWorkspace } from "@/hooks/use-workspaces";
// local components
import { WorkspaceDashboardHeader } from "./header";

function WorkspaceDashboardPage(): React.JSX.Element | null {
  const { workspaceSlug } = Route.useParams();
  const { data: currentWorkspace } = useCurrentWorkspace();
  const { t } = useTranslation();
  // derived values
  const pageTitle = currentWorkspace?.name ? `${currentWorkspace?.name} - ${t("home.title")}` : undefined;

  return (
    <HomeProvider key={workspaceSlug}>
      <AppHeader header={<WorkspaceDashboardHeader />} />
      <ContentWrapper>
        <PageHead title={pageTitle} />
        <WorkspaceHomeView />
      </ContentWrapper>
    </HomeProvider>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/")({
  component: WorkspaceDashboardPage,
});
