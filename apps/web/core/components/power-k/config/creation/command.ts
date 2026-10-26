/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCurrentWorkspace } from "@/hooks/use-workspaces";
import { FileText, FolderPlus, Layers, SquarePlus } from "lucide-react";
// plane imports
import { ContrastIcon, DiceIcon, LayersIcon } from "@plane/propel/icons";
// components
import { EUserProjectRoles, EUserWorkspaceRoles } from "@plane/types";
import type { TPowerKCommandConfig, TPowerKContext } from "@/components/power-k/core/types";
// hooks
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useParams } from "@tanstack/react-router";
import { useProjectPermissions, useWorkspacePermissions } from "@/hooks/use-permissions";
import { useCreationPermissions } from "@/hooks/use-creation-permissions";
import { useQuery } from "@tanstack/react-query";
import { instanceConfigOptions } from "@/queries/instance";

export type TPowerKCreationCommandKeys =
  | "create_work_item"
  | "create_page"
  | "create_view"
  | "create_cycle"
  | "create_module"
  | "create_project"
  | "create_workspace";

/**
 * Creation commands - Create any entity in the app
 */
export const usePowerKCreationCommandsRecord = (): Record<TPowerKCreationCommandKeys, TPowerKCommandConfig> => {
  // store
  const { data: config } = useQuery(instanceConfigOptions);
  const { canPerformAnyCreateAction } = useCreationPermissions();
  const { workspaceSlug, projectId } = useParams({ strict: false });
  const { can: canWorkspace } = useWorkspacePermissions(workspaceSlug?.toString());
  const { can: canProject } = useProjectPermissions(workspaceSlug?.toString(), projectId?.toString());
  const { data: catalogueWorkspace } = useCurrentWorkspace();
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  const workspaceProjectIds = projectRows
    ?.filter((project) => project.workspace === catalogueWorkspace?.id && !project.archived_at)
    .map((project) => project.id);
  const {
    toggleCreateIssueModal,
    toggleCreateProjectModal,
    toggleCreateCycleModal,
    toggleCreateModuleModal,
    toggleCreateViewModal,
    toggleCreatePageModal,
  } = useCommandPalette();
  // derived values
  const canCreateWorkItem = canPerformAnyCreateAction && workspaceProjectIds && workspaceProjectIds.length > 0;
  const canCreateProject = canWorkspace([EUserWorkspaceRoles.ADMIN, EUserWorkspaceRoles.MEMBER]);
  const hasProjectMemberLevelPermissions = (ctx: TPowerKContext) =>
    ctx.params.workspaceSlug?.toString() === workspaceSlug?.toString() &&
    ctx.params.projectId?.toString() === projectId?.toString() &&
    canProject([EUserProjectRoles.ADMIN, EUserProjectRoles.MEMBER]);
  const isWorkspaceCreationDisabled = config?.is_workspace_creation_disabled ?? false;

  const getProjectDetails = (ctx: TPowerKContext) =>
    ctx.params.projectId ? projectRows?.find((project) => project.id === ctx.params.projectId?.toString()) : undefined;

  return {
    create_work_item: {
      id: "create_work_item",
      type: "action",
      group: "create",
      i18n_title: "power_k.creation_actions.create_work_item",
      icon: LayersIcon,
      keySequence: "ni",
      action: () => toggleCreateIssueModal(true),
      isEnabled: () => Boolean(canCreateWorkItem),
      isVisible: () => Boolean(canCreateWorkItem),
      closeOnSelect: true,
    },
    create_page: {
      id: "create_page",
      type: "action",
      group: "create",
      i18n_title: "power_k.creation_actions.create_page",
      icon: FileText,
      keySequence: "nd",
      action: () => toggleCreatePageModal({ isOpen: true }),
      isEnabled: (ctx) => Boolean(getProjectDetails(ctx)?.page_view && hasProjectMemberLevelPermissions(ctx)),
      isVisible: (ctx) =>
        Boolean(ctx.params.projectId && getProjectDetails(ctx)?.page_view && hasProjectMemberLevelPermissions(ctx)),
      closeOnSelect: true,
    },
    create_view: {
      id: "create_view",
      type: "action",
      group: "create",
      i18n_title: "power_k.creation_actions.create_view",
      icon: Layers,
      keySequence: "nv",
      action: () => toggleCreateViewModal(true),
      isEnabled: (ctx) => Boolean(getProjectDetails(ctx)?.issue_views_view && hasProjectMemberLevelPermissions(ctx)),
      isVisible: (ctx) =>
        Boolean(
          ctx.params.projectId && getProjectDetails(ctx)?.issue_views_view && hasProjectMemberLevelPermissions(ctx)
        ),
      closeOnSelect: true,
    },
    create_cycle: {
      id: "create_cycle",
      type: "action",
      group: "create",
      i18n_title: "power_k.creation_actions.create_cycle",
      icon: ContrastIcon,
      keySequence: "nc",
      action: () => toggleCreateCycleModal(true),
      isEnabled: (ctx) => Boolean(getProjectDetails(ctx)?.cycle_view && hasProjectMemberLevelPermissions(ctx)),
      isVisible: (ctx) =>
        Boolean(ctx.params.projectId && getProjectDetails(ctx)?.cycle_view && hasProjectMemberLevelPermissions(ctx)),
      closeOnSelect: true,
    },
    create_module: {
      id: "create_module",
      type: "action",
      group: "create",
      i18n_title: "power_k.creation_actions.create_module",
      icon: DiceIcon,
      keySequence: "nm",
      action: () => toggleCreateModuleModal(true),
      isEnabled: (ctx) => Boolean(getProjectDetails(ctx)?.module_view && hasProjectMemberLevelPermissions(ctx)),
      isVisible: (ctx) =>
        Boolean(ctx.params.projectId && getProjectDetails(ctx)?.module_view && hasProjectMemberLevelPermissions(ctx)),
      closeOnSelect: true,
    },
    create_project: {
      id: "create_project",
      type: "action",
      group: "create",
      i18n_title: "power_k.creation_actions.create_project",
      icon: FolderPlus,
      keySequence: "np",
      action: () => toggleCreateProjectModal(true),
      isEnabled: () => Boolean(canCreateProject),
      isVisible: () => Boolean(canCreateProject),
      closeOnSelect: true,
    },
    create_workspace: {
      id: "create_workspace",
      type: "action",
      group: "create",
      i18n_title: "power_k.creation_actions.create_workspace",
      icon: SquarePlus,
      action: (ctx) => void ctx.router.navigate({ to: "/create-workspace" }),
      isEnabled: () => Boolean(!isWorkspaceCreationDisabled),
      isVisible: () => Boolean(!isWorkspaceCreationDisabled),
      closeOnSelect: true,
    },
  };
};
