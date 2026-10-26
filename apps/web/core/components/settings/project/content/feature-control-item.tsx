/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { notifyProjectFeatureUpdate } from "@/components/project/settings/feature-update-toast";
import type { IProject } from "@plane/types";
import { ToggleSwitch } from "@plane/ui";
// components
import { SettingsBoxedControlItem } from "@/components/settings/boxed-control-item";
// hooks
import { useProjectMutations } from "@/hooks/use-projects";

type Props = {
  description?: React.ReactNode;
  disabled?: boolean;
  projectId: string;
  featureProperty: keyof IProject;
  title: React.ReactNode;
  value: boolean;
  workspaceSlug: string;
};

export function ProjectSettingsFeatureControlItem(props: Props) {
  const { description, disabled, featureProperty, projectId, title, value, workspaceSlug } = props;
  const projectMutation = useProjectMutations(workspaceSlug, projectId);

  const handleSubmit = () => {
    if (!workspaceSlug || !projectId) return;

    // making the request to update the project feature
    const settingsPayload = {
      [featureProperty]: !value,
    };
    const updateProjectPromise = projectMutation
      .mutateAsync({
        type: "update",
        projectId,
        data: settingsPayload,
      })
      .then((result) => {
        if (!result.project) throw new Error("The server did not return the updated project.");
        return result.project;
      });

    notifyProjectFeatureUpdate(updateProjectPromise);
  };

  return (
    <SettingsBoxedControlItem
      title={title}
      description={description}
      control={
        <ToggleSwitch
          value={value}
          onChange={handleSubmit}
          disabled={disabled || projectMutation.isPending}
          size="sm"
        />
      }
    />
  );
}
