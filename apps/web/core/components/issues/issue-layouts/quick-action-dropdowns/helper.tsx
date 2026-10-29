/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useIssueAction } from "@/hooks/use-issue-action";
import { useCallback, useMemo } from "react";
import { useParams } from "@tanstack/react-router";
import { XCircle, ArchiveRestoreIcon } from "lucide-react";
// plane imports
import { useTranslation } from "@plane/i18n";
import { LinkIcon, CopyIcon, NewTabIcon, EditIcon, ArchiveIcon, TrashIcon } from "@plane/propel/icons";
import { ARCHIVABLE_STATE_GROUPS, EUserPermissions } from "@plane/constants";
import type { EIssuesStoreType, TIssue } from "@plane/types";
import type { TContextMenuItem } from "@plane/ui";
import { useWorkItemLinkCopy } from "@/hooks/use-work-item-link-copy";
import { generateWorkItemLink } from "@plane/utils/work-item";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useProjectStates } from "@/hooks/use-project-states";
import { useProjectPermissions } from "@/hooks/use-permissions";

export function useIssueQuickActionContext(issue: TIssue, readOnly: boolean) {
  const { workspaceSlug } = useParams({ strict: false });
  const { can: canProject } = useProjectPermissions(workspaceSlug, issue.project_id ?? undefined);
  const { getStateById } = useProjectStates(workspaceSlug, issue.project_id);
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  const stateDetails = getStateById(issue.state_id);

  return {
    workspaceSlug,
    projectIdentifier: projectRows?.find((project) => project.id === issue.project_id)?.identifier ?? "",
    isEditingAllowed: canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]) && !readOnly,
    isInArchivableGroup: !!stateDetails && ARCHIVABLE_STATE_GROUPS.includes(stateDetails.group),
  };
}

export interface MenuItemFactoryProps {
  issue: TIssue;
  workspaceSlug?: string;
  projectIdentifier?: string;
  isEditingAllowed: boolean;
  isArchivingAllowed?: boolean;
  isDeletingAllowed: boolean;
  isRestoringAllowed?: boolean;
  isInArchivableGroup?: boolean;
  issueTypeDetail?: { is_active?: boolean };
  // Action handlers
  setIssueToEdit: (issue: TIssue | undefined) => void;
  setCreateUpdateIssueModal: (open: boolean) => void;
  setDeleteIssueModal: (open: boolean) => void;
  setArchiveIssueModal?: (open: boolean) => void;
  onDuplicate?: () => void;
  handleRemoveFromView?: () => Promise<void>;
  handleRestore?: () => Promise<void>;
  // External handlers
  handleDelete?: () => Promise<void>;
  handleUpdate?: (data: TIssue) => Promise<void>;
  handleArchive?: () => Promise<void>;
  // Context-specific data
  cycleId?: string;
  moduleId?: string;
  storeType?: EIssuesStoreType;
}

// Common action handlers hook
const useIssueActionHandlers = (props: MenuItemFactoryProps) => {
  const { issue, workspaceSlug, projectIdentifier, handleRestore } = props;

  const workItemLink = useMemo(
    () =>
      generateWorkItemLink({
        workspaceSlug,
        projectId: issue?.project_id,
        issueId: issue?.id,
        projectIdentifier,
        sequenceId: issue?.sequence_id,
      }),
    [workspaceSlug, projectIdentifier, issue]
  );

  const copyLink = useWorkItemLinkCopy({ workspaceSlug, projectId: issue.project_id, issueId: issue.id }, workItemLink);

  const handleOpenInNewTab = () => window.open(workItemLink, "_blank");

  const restore = useIssueAction(
    { workspaceSlug, projectId: issue.project_id, issueId: issue.id, context: "restore" },
    props.isEditingAllowed && !!props.isRestoringAllowed && !!handleRestore,
    {
      success: "Your work item can be found in project work items.",
      error: "Work item could not be restored. Please try again.",
    }
  );
  const handleIssueRestore = () => {
    if (handleRestore) restore.run(handleRestore);
  };

  return {
    workItemLink,
    handleCopyIssueLink: copyLink.copy,
    copyDisabled: copyLink.disabled,
    handleOpenInNewTab,
    handleIssueRestore,
    restoreDisabled: restore.disabled,
  };
};

const useMenuItemFactory = (props: MenuItemFactoryProps) => {
  const { t } = useTranslation();
  const actionHandlers = useIssueActionHandlers(props);

  const {
    issue,
    isEditingAllowed,
    isArchivingAllowed = false,
    isDeletingAllowed,
    isRestoringAllowed = false,
    isInArchivableGroup = false,
    issueTypeDetail,
    setIssueToEdit,
    setCreateUpdateIssueModal,
    setDeleteIssueModal,
    setArchiveIssueModal,
    onDuplicate,
  } = props;

  const createEditMenuItem = (customEditAction?: () => void): TContextMenuItem => ({
    key: "edit",
    title: t("common.actions.edit"),
    icon: EditIcon,
    action:
      customEditAction ||
      (() => {
        setIssueToEdit(issue);
        setCreateUpdateIssueModal(true);
      }),
    shouldRender: isEditingAllowed,
  });

  const createCopyMenuItem = (): TContextMenuItem => ({
    key: "make-a-copy",
    title: t("common.actions.make_a_copy"),
    icon: CopyIcon,
    action: () => {
      if (onDuplicate) {
        onDuplicate();
        return;
      }
      setIssueToEdit(undefined);
      setCreateUpdateIssueModal(true);
    },
    shouldRender: isEditingAllowed && (issueTypeDetail?.is_active ?? true),
  });

  const createOpenInNewTabMenuItem = (): TContextMenuItem => ({
    key: "open-in-new-tab",
    title: t("common.actions.open_in_new_tab"),
    icon: NewTabIcon,
    action: actionHandlers.handleOpenInNewTab,
  });

  const createCopyLinkMenuItem = (): TContextMenuItem => ({
    key: "copy-link",
    title: t("common.actions.copy_link"),
    icon: LinkIcon,
    action: actionHandlers.handleCopyIssueLink,
    disabled: actionHandlers.copyDisabled,
  });

  const createArchiveMenuItem = (): TContextMenuItem => ({
    key: "archive",
    title: t("common.actions.archive"),
    description: isInArchivableGroup ? undefined : t("issue.archive.description"),
    icon: ArchiveIcon,
    className: "items-start",
    iconClassName: "mt-1",
    action: () => setArchiveIssueModal?.(true),
    disabled: !isInArchivableGroup || !setArchiveIssueModal,
    shouldRender: isArchivingAllowed,
  });

  const createRestoreMenuItem = (): TContextMenuItem => ({
    key: "restore",
    title: "Restore",
    icon: ArchiveRestoreIcon,
    action: actionHandlers.handleIssueRestore,
    disabled: actionHandlers.restoreDisabled,
    shouldRender: isRestoringAllowed,
  });

  const createDeleteMenuItem = (): TContextMenuItem => ({
    key: "delete",
    title: t("common.actions.delete"),
    icon: TrashIcon,
    action: () => {
      setDeleteIssueModal(true);
    },
    shouldRender: isDeletingAllowed,
  });

  return {
    ...actionHandlers,
    createEditMenuItem,
    createCopyMenuItem,
    createOpenInNewTabMenuItem,
    createCopyLinkMenuItem,
    createArchiveMenuItem,
    createRestoreMenuItem,
    createDeleteMenuItem,
    createPlanningMenuItems: (editAction: () => void, removeFromView: TContextMenuItem): TContextMenuItem[] => [
      createEditMenuItem(editAction),
      createCopyMenuItem(),
      createOpenInNewTabMenuItem(),
      createCopyLinkMenuItem(),
      removeFromView,
      createArchiveMenuItem(),
      createDeleteMenuItem(),
    ],
  };
};

function useRemoveFromViewMenuItem(props: MenuItemFactoryProps, kind: "cycle" | "module"): TContextMenuItem {
  const { issue, workspaceSlug, handleRemoveFromView, isEditingAllowed } = props;
  const { t } = useTranslation();
  const parentId = kind === "cycle" ? props.cycleId : props.moduleId;
  const removal = useIssueAction(
    {
      workspaceSlug,
      projectId: issue.project_id,
      issueId: issue.id,
      context: JSON.stringify(["remove-from-view", kind, parentId]),
    },
    isEditingAllowed && !!handleRemoveFromView && !!parentId,
    { error: t("entity.remove.failed", { entity: t("common.work_item") }) }
  );
  return {
    key: `remove-from-${kind}`,
    title: kind === "cycle" ? "Remove from cycle" : "Remove from module",
    icon: XCircle,
    action: () => {
      if (handleRemoveFromView) removal.run(handleRemoveFromView);
    },
    disabled: removal.disabled,
    shouldRender: isEditingAllowed,
  };
}

// Predefined menu item sets for different contexts
export const useIssueMenuItems = (props: MenuItemFactoryProps): TContextMenuItem[] => {
  const factory = useMenuItemFactory(props);

  return [
    factory.createEditMenuItem(),
    factory.createCopyMenuItem(),
    factory.createOpenInNewTabMenuItem(),
    factory.createCopyLinkMenuItem(),
    factory.createArchiveMenuItem(),
    factory.createDeleteMenuItem(),
  ];
};

export const useWorkItemDetailMenuItems = (props: MenuItemFactoryProps): TContextMenuItem[] => {
  const factory = useMenuItemFactory(props);

  return [
    factory.createCopyMenuItem(),
    factory.createOpenInNewTabMenuItem(),
    factory.createArchiveMenuItem(),
    factory.createRestoreMenuItem(),
    factory.createDeleteMenuItem(),
  ];
};

export const useCycleIssueMenuItems = (props: MenuItemFactoryProps): TContextMenuItem[] => {
  const factory = useMenuItemFactory(props);
  const removeFromView = useRemoveFromViewMenuItem(props, "cycle");
  const { cycleId, issue, setCreateUpdateIssueModal, setIssueToEdit } = props;

  const customEditAction = useCallback(() => {
    setIssueToEdit({
      ...issue,
      cycle_id: cycleId ?? null,
    });
    setCreateUpdateIssueModal(true);
  }, [cycleId, issue, setCreateUpdateIssueModal, setIssueToEdit]);

  return factory.createPlanningMenuItems(customEditAction, removeFromView);
};

export const useModuleIssueMenuItems = (props: MenuItemFactoryProps): TContextMenuItem[] => {
  const factory = useMenuItemFactory(props);
  const removeFromView = useRemoveFromViewMenuItem(props, "module");
  const { issue, moduleId, setCreateUpdateIssueModal, setIssueToEdit } = props;

  const customEditAction = useCallback(() => {
    setIssueToEdit({
      ...issue,
      module_ids: moduleId ? [moduleId] : [],
    });
    setCreateUpdateIssueModal(true);
  }, [issue, moduleId, setCreateUpdateIssueModal, setIssueToEdit]);

  return factory.createPlanningMenuItems(customEditAction, removeFromView);
};

export const useArchivedIssueMenuItems = (props: MenuItemFactoryProps): TContextMenuItem[] => {
  const factory = useMenuItemFactory(props);

  return [
    factory.createRestoreMenuItem(),
    factory.createOpenInNewTabMenuItem(),
    factory.createCopyLinkMenuItem(),
    factory.createDeleteMenuItem(),
  ];
};
