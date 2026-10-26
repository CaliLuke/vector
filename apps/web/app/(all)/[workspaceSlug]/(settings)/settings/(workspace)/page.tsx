/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute } from "@tanstack/react-router";

// plane imports
import { useTranslation } from "@plane/i18n";
// components
import { PageHead } from "@/components/core/page-title";
import { SettingsContentWrapper } from "@/components/settings/content-wrapper";
import { WorkspaceDetails } from "@/components/workspace/settings/workspace-details";
// hooks
import { useCurrentWorkspace } from "@/hooks/use-workspaces";
// local imports
import { GeneralWorkspaceSettingsHeader } from "./header";

function GeneralWorkspaceSettingsPage(): React.JSX.Element | null {
  // store hooks
  const { data: currentWorkspace } = useCurrentWorkspace();
  const { t } = useTranslation();
  // derived values
  const pageTitle = currentWorkspace?.name
    ? t("workspace_settings.page_label", { workspace: currentWorkspace.name })
    : undefined;

  return (
    <SettingsContentWrapper header={<GeneralWorkspaceSettingsHeader />}>
      <PageHead title={pageTitle} />
      <WorkspaceDetails />
    </SettingsContentWrapper>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/settings/_workspace/")({
  component: GeneralWorkspaceSettingsPage,
});
