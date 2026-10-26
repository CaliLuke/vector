/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute } from "@tanstack/react-router";
import { ProjectFeatureSettings } from "@/components/settings/project/feature";
import { FeaturesCyclesProjectSettingsHeader } from "./header";

function FeaturesCyclesSettingsPage() {
  const { workspaceSlug, projectId } = Route.useParams();
  return (
    <ProjectFeatureSettings
      workspaceSlug={workspaceSlug}
      projectId={projectId}
      feature="cycles"
      header={<FeaturesCyclesProjectSettingsHeader />}
    />
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/settings/projects/$projectId/features/cycles")({
  component: FeaturesCyclesSettingsPage,
});
