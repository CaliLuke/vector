/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useEffect, useRef } from "react";

import {
  Bell,
  BellOff,
  Signal,
  TagIcon,
  TicketCheck,
  Triangle,
  Type,
  UserMinus2,
  UserPlus2,
  Users,
} from "lucide-react";
// plane imports
import { useTranslation } from "@plane/i18n";
import { LinkIcon, TrashIcon, ContrastIcon, DiceIcon, DoubleCircleIcon } from "@plane/propel/icons";
import { setToast, TOAST_TYPE } from "@plane/propel/toast";
import type { ICycle, IIssueLabel, IModule, TIssue, TIssuePriorities } from "@plane/types";
import { EIssueServiceType, EIssuesStoreType, EUserProjectRoles } from "@plane/types";
import { copyTextToClipboard } from "@plane/utils/string";
// components
import type { TPowerKCommandConfig } from "@/components/power-k/core/types";
// hooks
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useParams } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { issueSubscriptionOptions, updateIssueSubscriptionOptions } from "@/queries/issue-subscription";
import { useWorkItemByIdentifier } from "@/hooks/use-work-item-detail";
import { useIssueListMutationOwner } from "@/hooks/use-issue-list-mutation-owner";
import { useWorkItemEntityMutationActions, workItemProjectDetails } from "./entity-record";

type TWorkItemCycleAction = (
  workspaceSlug: string,
  projectId: string,
  cycleId: string,
  issueId: string
) => Promise<void>;
type TWorkItemModuleAction = (
  workspaceSlug: string,
  projectId: string,
  issueId: string,
  add: string[],
  remove: string[]
) => Promise<void>;
type TWorkItemCommandContext = {
  workspaceSlug: string | string[] | undefined;
  entityDetails: TIssue | null | undefined;
  projectDetails: { cycle_view?: boolean; module_view?: boolean } | undefined;
  currentUser: { id: string } | null | undefined;
  isEpic: boolean;
  isCurrentUserAssigned: boolean;
  isEstimateEnabled: boolean;
  isEditingAllowed: boolean;
  isSubscribed: boolean | undefined;
  subscriptionPending: boolean;
  handleUpdateEntity: (data: Partial<TIssue>) => void;
  handleUpdateAssignee: (assigneeId: string) => void;
  addCycleToIssue: TWorkItemCycleAction;
  removeIssueFromCycle: TWorkItemCycleAction;
  changeModulesInIssue: TWorkItemModuleAction;
  addCycleToEpic: TWorkItemCycleAction;
  removeEpicFromCycle: TWorkItemCycleAction;
  changeModulesInEpic: TWorkItemModuleAction;
  handleSubscription: () => void;
  handleDeleteWorkItem: () => void;
  copyWorkItemIdToClipboard: () => void;
  copyWorkItemTitleToClipboard: () => void;
  copyWorkItemUrlToClipboard: () => void;
};

function buildBasicCommands(
  context: Pick<
    TWorkItemCommandContext,
    | "entityDetails"
    | "handleUpdateEntity"
    | "isEditingAllowed"
    | "handleUpdateAssignee"
    | "isCurrentUserAssigned"
    | "currentUser"
    | "isEstimateEnabled"
  >
): TPowerKCommandConfig[] {
  const {
    entityDetails,
    handleUpdateEntity,
    isEditingAllowed,
    handleUpdateAssignee,
    isCurrentUserAssigned,
    currentUser,
    isEstimateEnabled,
  } = context;
  return [
    {
      id: "change_work_item_state",
      i18n_title: "power_k.contextual_actions.work_item.change_state",
      icon: DoubleCircleIcon,
      group: "contextual",
      contextType: "work-item",
      type: "change-page",
      page: "update-work-item-state",
      onSelect: (data) => {
        const stateId = data as string;
        if (entityDetails?.state_id === stateId) return;
        handleUpdateEntity({
          state_id: stateId,
        });
      },
      shortcut: "s",
      isEnabled: () => isEditingAllowed,
      isVisible: () => isEditingAllowed,
      closeOnSelect: true,
    },
    {
      id: "change_work_item_priority",
      i18n_title: "power_k.contextual_actions.work_item.change_priority",
      icon: Signal,
      group: "contextual",
      contextType: "work-item",
      type: "change-page",
      page: "update-work-item-priority",
      onSelect: (data) => {
        const priority = data as TIssuePriorities;
        if (entityDetails?.priority === priority) return;
        handleUpdateEntity({
          priority,
        });
      },
      shortcut: "p",
      isEnabled: () => isEditingAllowed,
      isVisible: () => isEditingAllowed,
      closeOnSelect: true,
    },
    {
      id: "change_work_item_assignees",
      i18n_title: "power_k.contextual_actions.work_item.change_assignees",
      icon: Users,
      group: "contextual",
      contextType: "work-item",
      type: "change-page",
      page: "update-work-item-assignee",
      onSelect: (data) => {
        const assigneeId = data as string;
        handleUpdateAssignee(assigneeId);
      },
      shortcut: "a",
      isEnabled: () => isEditingAllowed,
      isVisible: () => isEditingAllowed,
      closeOnSelect: false,
    },
    {
      id: "assign_work_item_to_me",
      i18n_title: isCurrentUserAssigned
        ? "power_k.contextual_actions.work_item.unassign_from_me"
        : "power_k.contextual_actions.work_item.assign_to_me",
      icon: isCurrentUserAssigned ? UserMinus2 : UserPlus2,
      group: "contextual",
      contextType: "work-item",
      type: "action",
      action: () => {
        if (!currentUser) return;
        handleUpdateAssignee(currentUser.id);
      },
      shortcut: "i",
      isEnabled: () => isEditingAllowed,
      isVisible: () => isEditingAllowed,
      closeOnSelect: true,
    },
    {
      id: "change_work_item_estimate",
      i18n_title: "power_k.contextual_actions.work_item.change_estimate",
      icon: Triangle,
      group: "contextual",
      contextType: "work-item",
      type: "change-page",
      page: "update-work-item-estimate",
      onSelect: (data) => {
        const estimatePointId = data as string | null;
        if (entityDetails?.estimate_point === estimatePointId) return;
        handleUpdateEntity({
          estimate_point: estimatePointId,
        });
      },
      modifierShortcut: "shift+e",
      isEnabled: () => isEstimateEnabled && isEditingAllowed,
      isVisible: () => isEstimateEnabled && isEditingAllowed,
      closeOnSelect: true,
    },
  ];
}

function buildRelationshipCommands(
  context: Pick<
    TWorkItemCommandContext,
    | "workspaceSlug"
    | "entityDetails"
    | "addCycleToIssue"
    | "removeIssueFromCycle"
    | "changeModulesInIssue"
    | "addCycleToEpic"
    | "removeEpicFromCycle"
    | "changeModulesInEpic"
    | "projectDetails"
    | "isEditingAllowed"
  >
): TPowerKCommandConfig[] {
  const {
    workspaceSlug,
    entityDetails,
    addCycleToIssue,
    removeIssueFromCycle,
    changeModulesInIssue,
    addCycleToEpic,
    removeEpicFromCycle,
    changeModulesInEpic,
    projectDetails,
    isEditingAllowed,
  } = context;
  return [
    {
      id: "add_work_item_to_cycle",
      i18n_title: "power_k.contextual_actions.work_item.add_to_cycle",
      icon: ContrastIcon,
      group: "contextual",
      contextType: "work-item",
      type: "change-page",
      page: "update-work-item-cycle",
      onSelect: async (data) => {
        const cycleId = (data as ICycle)?.id;
        if (!workspaceSlug || !entityDetails || !entityDetails.project_id) return;
        if (entityDetails.cycle_id === cycleId) return;
        // handlers
        const addCycleToEntity = entityDetails.is_epic ? addCycleToEpic : addCycleToIssue;
        const removeCycleFromEntity = entityDetails.is_epic ? removeEpicFromCycle : removeIssueFromCycle;

        try {
          if (cycleId) {
            await addCycleToEntity(workspaceSlug.toString(), entityDetails.project_id, cycleId, entityDetails.id);
          } else {
            await removeCycleFromEntity(
              workspaceSlug.toString(),
              entityDetails.project_id,
              entityDetails.cycle_id ?? "",
              entityDetails.id
            );
          }
        } catch {
          setToast({
            type: TOAST_TYPE.ERROR,
            title: "Error!",
            message: `${entityDetails.is_epic ? "Epic" : "Work item"} could not be updated. Please try again.`,
          });
        }
      },
      modifierShortcut: "shift+c",
      isEnabled: () => Boolean(projectDetails?.cycle_view && isEditingAllowed),
      isVisible: () => Boolean(projectDetails?.cycle_view && isEditingAllowed),
      closeOnSelect: true,
    },
    {
      id: "add_work_item_to_modules",
      i18n_title: "power_k.contextual_actions.work_item.add_to_modules",
      icon: DiceIcon,
      group: "contextual",
      contextType: "work-item",
      type: "change-page",
      page: "update-work-item-module",
      onSelect: async (data) => {
        const moduleId = (data as IModule)?.id;
        if (!workspaceSlug || !entityDetails || !entityDetails.project_id) return;
        // handlers
        const changeModulesInEntity = entityDetails.is_epic ? changeModulesInEpic : changeModulesInIssue;
        try {
          if (entityDetails.module_ids?.includes(moduleId)) {
            await changeModulesInEntity(
              workspaceSlug.toString(),
              entityDetails.project_id,
              entityDetails.id,
              [],
              [moduleId]
            );
          } else {
            await changeModulesInEntity(
              workspaceSlug.toString(),
              entityDetails.project_id,
              entityDetails.id,
              [moduleId],
              []
            );
          }
        } catch {
          setToast({
            type: TOAST_TYPE.ERROR,
            title: "Error!",
            message: `${entityDetails.is_epic ? "Epic" : "Work item"} could not be updated. Please try again.`,
          });
        }
      },
      modifierShortcut: "shift+m",
      isEnabled: () => Boolean(projectDetails?.module_view && isEditingAllowed),
      isVisible: () => Boolean(projectDetails?.module_view && isEditingAllowed),
      closeOnSelect: false,
    },
  ];
}

function buildLabelCommands(
  context: Pick<TWorkItemCommandContext, "workspaceSlug" | "entityDetails" | "handleUpdateEntity" | "isEditingAllowed">
): TPowerKCommandConfig[] {
  const { workspaceSlug, entityDetails, handleUpdateEntity, isEditingAllowed } = context;
  return [
    {
      id: "add_work_item_labels",
      i18n_title: "power_k.contextual_actions.work_item.add_labels",
      icon: TagIcon,
      group: "contextual",
      contextType: "work-item",
      type: "change-page",
      page: "update-work-item-labels",
      onSelect: (data) => {
        const labelId = (data as IIssueLabel)?.id;
        if (!workspaceSlug || !entityDetails || !entityDetails.project_id) return;
        const updatedLabels = [...(entityDetails.label_ids ?? [])];
        if (updatedLabels.includes(labelId)) updatedLabels.splice(updatedLabels.indexOf(labelId), 1);
        else updatedLabels.push(labelId);
        handleUpdateEntity({
          label_ids: updatedLabels,
        });
      },
      shortcut: "l",
      isEnabled: () => isEditingAllowed,
      isVisible: () => isEditingAllowed,
      closeOnSelect: false,
    },
  ];
}

function buildUtilityCommands(
  context: Pick<
    TWorkItemCommandContext,
    | "isSubscribed"
    | "subscriptionPending"
    | "handleSubscription"
    | "handleDeleteWorkItem"
    | "copyWorkItemIdToClipboard"
    | "copyWorkItemTitleToClipboard"
    | "copyWorkItemUrlToClipboard"
    | "isEditingAllowed"
  >
): TPowerKCommandConfig[] {
  const {
    isSubscribed,
    subscriptionPending,
    handleSubscription,
    handleDeleteWorkItem,
    copyWorkItemIdToClipboard,
    copyWorkItemTitleToClipboard,
    copyWorkItemUrlToClipboard,
    isEditingAllowed,
  } = context;
  return [
    {
      id: "subscribe_work_item",
      i18n_title: isSubscribed
        ? "power_k.contextual_actions.work_item.unsubscribe"
        : "power_k.contextual_actions.work_item.subscribe",
      icon: isSubscribed ? BellOff : Bell,
      group: "contextual",
      contextType: "work-item",
      type: "action",
      action: handleSubscription,
      modifierShortcut: "shift+s",
      isEnabled: () => isEditingAllowed && isSubscribed !== undefined && !subscriptionPending,
      isVisible: () => isEditingAllowed,
      closeOnSelect: true,
    },
    {
      id: "delete_work_item",
      i18n_title: "power_k.contextual_actions.work_item.delete",
      icon: TrashIcon,
      group: "contextual",
      contextType: "work-item",
      type: "action",
      action: handleDeleteWorkItem,
      modifierShortcut: "cmd+backspace",
      isEnabled: () => isEditingAllowed,
      isVisible: () => isEditingAllowed,
      closeOnSelect: true,
    },
    {
      id: "copy_work_item_id",
      i18n_title: "power_k.contextual_actions.work_item.copy_id",
      icon: TicketCheck,
      group: "contextual",
      contextType: "work-item",
      type: "action",
      action: copyWorkItemIdToClipboard,
      modifierShortcut: "cmd+.",
      isEnabled: () => true,
      isVisible: () => true,
      closeOnSelect: true,
    },
    {
      id: "copy_work_item_title",
      i18n_title: "power_k.contextual_actions.work_item.copy_title",
      icon: Type,
      group: "contextual",
      contextType: "work-item",
      type: "action",
      action: copyWorkItemTitleToClipboard,
      modifierShortcut: "cmd+shift+'",
      isEnabled: () => true,
      isVisible: () => true,
      closeOnSelect: true,
    },
    {
      id: "copy_work_item_url",
      i18n_title: "power_k.contextual_actions.work_item.copy_url",
      icon: LinkIcon,
      group: "contextual",
      contextType: "work-item",
      type: "action",
      action: copyWorkItemUrlToClipboard,
      modifierShortcut: "cmd+shift+,",
      isEnabled: () => true,
      isVisible: () => true,
      closeOnSelect: true,
    },
  ];
}

function useWorkItemEntityQueries() {
  const { workspaceSlug, workItem: entityIdentifier } = useParams({ strict: false });
  const projectCatalogue = useWorkspaceProjectDetails(workspaceSlug?.toString());
  const [projectIdentifier, sequenceId] = (entityIdentifier?.toString() ?? "").split("-");
  const identifierProjectId = projectCatalogue.projects?.find(
    (project) => project.identifier === projectIdentifier
  )?.id;
  const issueDetail = useWorkItemByIdentifier(
    workspaceSlug?.toString(),
    projectIdentifier,
    sequenceId,
    identifierProjectId,
    EIssueServiceType.ISSUES
  );
  const epicDetail = useWorkItemByIdentifier(
    workspaceSlug?.toString(),
    projectIdentifier,
    sequenceId,
    identifierProjectId,
    EIssueServiceType.EPICS
  );
  return { workspaceSlug, projectRows: projectCatalogue.projects, issueDetail, epicDetail };
}

function useWorkItemEntityRecord() {
  const client = useQueryClient();
  const { data: currentUser } = useCurrentUser();
  const { toggleDeleteIssueModal } = useCommandPalette();
  const { workspaceSlug, projectRows, issueDetail, epicDetail } = useWorkItemEntityQueries();
  const issueDetails = issueDetail.issue;
  const epicDetails = epicDetail.issue;
  const entityDetails = issueDetails ?? epicDetails;
  const entityId = entityDetails?.id;
  const isEpic = Boolean(entityDetails?.is_epic);
  const detailScope = isEpic ? epicDetail.scope : issueDetail.scope;
  const mutationStoreType = isEpic ? EIssuesStoreType.EPIC : EIssuesStoreType.PROJECT;
  const resolveProjectListOwner = useIssueListMutationOwner(mutationStoreType);
  const { projectDetails, isEstimateEnabled } = workItemProjectDetails(entityDetails, projectRows);
  const isCurrentUserAssigned = Boolean(entityDetails?.assignee_ids?.includes(currentUser?.id ?? ""));
  const mutations = useWorkItemEntityMutationActions(client, detailScope, resolveProjectListOwner);

  return {
    workspaceSlug,
    entityId,
    entityDetails,
    isEpic,
    projectDetails,
    isCurrentUserAssigned,
    isEstimateEnabled,
    currentUser,
    toggleDeleteIssueModal,
    ...mutations,
  };
}

function useWorkItemSubscription(data: ReturnType<typeof useWorkItemEntityRecord>) {
  const client = useQueryClient();
  const subscriptionScope = {
    userId: data.currentUser?.id,
    workspaceSlug: data.workspaceSlug,
    projectId: data.entityDetails?.project_id ?? undefined,
    issueId: data.entityId ?? undefined,
    serviceType: data.isEpic ? EIssueServiceType.EPICS : EIssueServiceType.ISSUES,
  };
  const subscription = useQuery(issueSubscriptionOptions(subscriptionScope));
  const {
    mutate: updateSubscription,
    reset: resetSubscription,
    isPending: subscriptionPending,
  } = useMutation(updateIssueSubscriptionOptions(client, subscriptionScope));
  const entityId = data.entityId;
  const workspaceSlug = data.workspaceSlug;
  const previousScope = useRef({ entityId, workspaceSlug });
  useEffect(() => {
    if (previousScope.current.entityId !== entityId || previousScope.current.workspaceSlug !== workspaceSlug) {
      resetSubscription();
      previousScope.current = { entityId, workspaceSlug };
    }
  }, [entityId, workspaceSlug, resetSubscription]);
  return { isSubscribed: subscription.data, subscriptionPending, updateSubscription };
}

function useWorkItemCommandData() {
  const record = useWorkItemEntityRecord();
  const { can } = useProjectPermissions(
    record.workspaceSlug?.toString() ?? undefined,
    record.entityDetails?.project_id ?? undefined
  );
  const isEditingAllowed =
    can([EUserProjectRoles.ADMIN, EUserProjectRoles.MEMBER]) && !record.entityDetails?.archived_at;
  const subscription = useWorkItemSubscription(record);
  return { ...record, ...subscription, isEditingAllowed };
}

function useWorkItemCommandHandlers(data: ReturnType<typeof useWorkItemCommandData>) {
  const { t } = useTranslation();
  const { workspaceSlug, entityDetails, updateEntity, isEpic, isSubscribed, subscriptionPending, updateSubscription } =
    data;
  const { toggleDeleteIssueModal } = data;
  const handleUpdateEntity = useCallback(
    async (formData: Partial<TIssue>) => {
      if (!workspaceSlug || !entityDetails || !entityDetails.project_id) return;
      await updateEntity(workspaceSlug, entityDetails.project_id, entityDetails.id, formData).catch(() => {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message: `${isEpic ? "Epic" : "Work item"} could not be updated. Please try again.`,
        });
      });
    },
    [workspaceSlug, entityDetails, updateEntity, isEpic]
  );
  const handleUpdateAssignee = useCallback(
    (assigneeId: string) => {
      if (!entityDetails) return;
      const updatedAssignees = [...(entityDetails.assignee_ids ?? [])];
      if (updatedAssignees.includes(assigneeId)) updatedAssignees.splice(updatedAssignees.indexOf(assigneeId), 1);
      else updatedAssignees.push(assigneeId);
      void handleUpdateEntity({ assignee_ids: updatedAssignees });
    },
    [entityDetails, handleUpdateEntity]
  );
  const handleSubscription = useCallback(() => {
    if (isSubscribed === undefined || subscriptionPending) return;
    updateSubscription(!isSubscribed, {
      onSuccess: () =>
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("toast.success"),
          message: isSubscribed
            ? t("issue.subscription.actions.unsubscribed")
            : t("issue.subscription.actions.subscribed"),
        }),
      onError: () => setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t("common.error.message") }),
    });
  }, [isSubscribed, subscriptionPending, updateSubscription, t]);
  const handleDeleteWorkItem = useCallback(() => toggleDeleteIssueModal(true), [toggleDeleteIssueModal]);
  return { handleUpdateEntity, handleUpdateAssignee, handleSubscription, handleDeleteWorkItem };
}

function useWorkItemClipboardActions(data: ReturnType<typeof useWorkItemCommandData>) {
  const { t } = useTranslation();
  const { sequence_id: sequenceId, name: workItemName } = data.entityDetails ?? {};
  const identifier = data.projectDetails?.identifier;
  const copyWorkItemIdToClipboard = useCallback(() => {
    const id = `${identifier}-${sequenceId}`;
    void copyTextToClipboard(id)
      .then(() => {
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("power_k.contextual_actions.work_item.copy_id_toast_success"),
        });
        return undefined;
      })
      .catch(() => {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("power_k.contextual_actions.work_item.copy_id_toast_error"),
        });
        return undefined;
      });
  }, [sequenceId, identifier, t]);
  const copyWorkItemTitleToClipboard = useCallback(() => {
    void copyTextToClipboard(workItemName ?? "")
      .then(() => {
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("power_k.contextual_actions.work_item.copy_title_toast_success"),
        });
        return undefined;
      })
      .catch(() => {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("power_k.contextual_actions.work_item.copy_title_toast_error"),
        });
        return undefined;
      });
  }, [workItemName, t]);
  const copyWorkItemUrlToClipboard = useCallback(() => {
    void copyTextToClipboard(new URL(window.location.href).href)
      .then(() => {
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("power_k.contextual_actions.work_item.copy_url_toast_success"),
        });
        return undefined;
      })
      .catch(() => {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("power_k.contextual_actions.work_item.copy_url_toast_error"),
        });
        return undefined;
      });
  }, [t]);
  return { copyWorkItemIdToClipboard, copyWorkItemTitleToClipboard, copyWorkItemUrlToClipboard };
}

export const usePowerKWorkItemContextBasedCommands = (): TPowerKCommandConfig[] => {
  const data = useWorkItemCommandData();
  const handlers = useWorkItemCommandHandlers(data);
  const clipboardActions = useWorkItemClipboardActions(data);
  const commandContext: TWorkItemCommandContext = { ...data, ...handlers, ...clipboardActions };
  return [
    ...buildBasicCommands(commandContext),
    ...buildRelationshipCommands(commandContext),
    ...buildLabelCommands(commandContext),
    ...buildUtilityCommands(commandContext),
  ];
};
