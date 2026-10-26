/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQueryClient } from "@tanstack/react-query";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { currentParentCatalogueScope } from "@/queries/cycle-module-scope";
import { captureProjectAccessScope } from "@/queries/project-access-scope";
import { useParams } from "@tanstack/react-router";
import { useRef, useState } from "react";
import type { ReactNode } from "react";

// Vector imports
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { TIssue } from "@plane/types";
import { EIssueServiceType, EIssuesStoreType } from "@plane/types";
import { EModalPosition, EModalWidth, ModalCore } from "@plane/ui";
// hooks
import { useIssueModal } from "@/hooks/context/use-issue-modal";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useWorkspaceDraftMutations } from "@/hooks/use-workspace-drafts";
import { useReviewedItemEdit } from "./use-reviewed-item-edit";
import { ItemEditRecovery, ItemEditConfirmation } from "./item-edit-confirmation";
import { itemEditChanges, workItemEditsService } from "@/services/work-item-edits.service";
import type { ItemEditCommand } from "@/services/work-item-edits.service";
import { useIssueModalSource } from "./use-issue-modal-source";
import { useIssueListType } from "@/hooks/use-issue-list-type";
import { useIssuesActions } from "@/hooks/use-issues-actions";
import { useWorkItemDetailScope } from "@/hooks/use-work-item-detail";
import { captureIssueDetailReconciliation } from "@/queries/issue-details";
import { useIssueListMutationOwner } from "@/hooks/use-issue-list-mutation-owner";
import { captureIssueListMutationOwner, refreshCapturedIssueLists } from "@/queries/issue-list-mutations";
// services
import { FileService } from "@/services/file.service";
const fileService = new FileService();
// local imports
import { CreateIssueToastActionItems } from "../create-issue-toast-action-items";
import { DraftIssueLayout } from "./draft-issue-layout";
import { IssueFormRoot } from "./form";
import { IssueCreationIncompleteError, useIssueCreation } from "./use-issue-creation";
import type { CreatedWorkItem } from "./use-issue-creation";
import type { IssueFormProps } from "./form";
import type { IssuesModalProps } from "./modal";

export function CreateUpdateIssueModalBase(props: IssuesModalProps) {
  const {
    data,
    isOpen,
    onClose,
    beforeFormSubmit,
    onSubmit,
    withDraftIssueWrapper = true,
    storeType: issueStoreFromProps,
    isDraft = false,
    fetchIssueDetails = true,
    moveToIssue = false,
    modalTitle,
    primaryButtonText,
    isProjectSelectionDisabled = false,
    showActionItemsOnUpdate = false,
  } = props;
  const issueStoreType = useIssueListType();
  const client = useQueryClient(),
    accountId = useCurrentUserId(),
    generation = useIdentityGeneration(client);

  const { storeType, mutationStoreType } = issueModalStoreTypes(issueStoreFromProps, issueStoreType);
  // ref
  const issueTitleRef = useRef<HTMLInputElement>(null);
  // states
  const [changesMade, setChangesMade] = useState<Partial<TIssue> | null>(null);
  const [sourceGeneration, setSourceGeneration] = useState(0);
  const [createMore, setCreateMore] = useState(false);
  const [uploadedAssetIds, setUploadedAssetIds] = useState<string[]>([]);
  // store hooks
  const { t } = useTranslation();
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const { projectId: routerProjectId, cycleId, moduleId, workItem } = useParams({ strict: false });
  const creationScope = JSON.stringify([
    accountId,
    generation,
    workspaceSlug,
    routerProjectId,
    cycleId,
    moduleId,
    workItem,
    data?.id,
    isOpen,
    sourceGeneration,
  ]);
  const creationCurrent = useCommittedScope(creationScope);
  const draftMutations = useWorkspaceDraftMutations(workspaceSlug);
  const { allowedProjectIds, defaultProjectId } = useIssueModal();
  const projectCatalogue = useWorkspaceProjectDetails(workspaceSlug);
  const projectRows = projectCatalogue.projects;
  // current store details
  const actions = useIssuesActions(mutationStoreType);
  const projectActions = useIssuesActions(EIssuesStoreType.PROJECT);
  const resolveListOwner = useIssueListMutationOwner(mutationStoreType);
  const edits = useReviewedItemEdit(workspaceSlug, data?.id, {
    captureAccess: () => {
      return () => {
        if (!sourceAccessAllowed || !editAccessCurrent())
          throw new Error("Work item access changed. Reload before saving.");
      };
    },
    preview: (command, beforeSend) => workItemEditsService.preview(workspaceSlug, command, beforeSend),
    commit: (command, beforeSend) => workItemEditsService.commit(workspaceSlug, command, beforeSend),
  });
  const {
    activeProjectId,
    descriptionQuery,
    sourceVersion,
    accessState,
    retryAccess,
    sourceAccessAllowed,
    accessKey,
    description,
    fields: savedFields,
  } = useIssueModalSource({
    data,
    isOpen,
    fetchIssueDetails,
    workspaceSlug,
    routerProjectId,
    workItem,
    defaultProjectId,
    getProjectByIdentifier: (identifier) => projectRows?.find((project) => project.identifier === identifier),
    isDraft,
    retainedPublication: edits.recoveryCommand?.changes,
  });
  const editAccessCurrent = useCommittedScope(JSON.stringify([sourceAccessAllowed, accessKey]));
  const detailServiceType = issueModalServiceType(issueStoreFromProps, issueStoreType);
  const detailMutationAccess = useWorkItemDetailScope(
    workspaceSlug,
    issueDetailProjectId(activeProjectId, data?.project_id, routerProjectId),
    detailServiceType,
    data?.id
  );

  const addIssueToCycle = async (issue: { id: string; project_id?: string | null }, targetCycleId: string) => {
    if (!issue.project_id) return;

    await actions.addCycleToIssue(issue.project_id, targetCycleId, issue.id);
  };

  const addIssueToModule = async (issue: { id: string; project_id?: string | null }, moduleIds: string[]) => {
    if (!issue.project_id) return;

    await actions.changeModulesInIssue(issue.project_id, issue.id, moduleIds, []);
  };

  const handleCreateMoreToggleChange = (value: boolean) => {
    setCreateMore(value);
  };

  const handleClose = () => {
    setChangesMade(null);
    onClose();
  };

  async function finalizeAssetUploads(
    entityId: string,
    projectId: string | null | undefined,
    draft: boolean,
    references: string[]
  ) {
    const assetIds = uploadedAssetIds.filter((id) => references.includes(id));
    if (!assetIds.length) return;
    if (draft)
      await fileService.updateBulkWorkspaceAssetsUploadStatus(workspaceSlug, entityId, { asset_ids: assetIds });
    else {
      if (!projectId) throw new Error("Select a project.");
      await fileService.updateBulkProjectAssetsUploadStatus(workspaceSlug, projectId, entityId, {
        asset_ids: assetIds,
      });
    }
    setUploadedAssetIds((current) => current.filter((id) => !assetIds.includes(id)));
  }

  const saveReviewedChange = async (command: ItemEditCommand, draft?: Partial<TIssue>) => {
    const receiptOwner =
      command.mode === "publish" || command.mode === "draft"
        ? draftMutations.captureReceiptOwner(command.item_id)
        : undefined;
    const detailReceiptOwner = captureEditReceiptOwner(client, command, detailMutationAccess);
    const projectIds = editedIssueProjectIds(data?.project_id, command.project_id);
    const capturedLists = await captureEditedIssueLists(client, resolveListOwner, workspaceSlug, projectIds);
    const receipt = await edits.submit(command, draft);
    if (!edits.isCurrent()) return;
    if (command.mode === "publish") await draftMutations.completePublication(command.item_id, receiptOwner);
    if ("sequence_id" in receipt) {
      await detailReceiptOwner?.reconcile(receipt);
      await Promise.all(capturedLists.map((captured) => refreshCapturedIssueLists(client, captured)));
      await onSubmit?.(receipt);
    } else await draftMutations.acceptDraftReceipt(receipt, receiptOwner);
    if (!edits.isCurrent()) return;
    edits.acknowledge();
    setToast({
      type: TOAST_TYPE.SUCCESS,
      title: t("success"),
      message: t(command.mode === "publish" ? "issue_created_successfully" : "issue_updated_successfully"),
      actionItems: updateActionItems(
        workspaceSlug,
        receipt.id,
        receipt.project_id,
        command.mode === "publish" || showActionItemsOnUpdate
      ),
    });
    handleClose();
  };
  const editCommand = (
    payload: Partial<TIssue>,
    mode: ItemEditCommand["mode"],
    revision?: number,
    version?: string
  ): ItemEditCommand => {
    if (!data?.id) throw new Error("Select a work item to save.");
    if (revision === undefined) throw new Error("Wait for the work item configuration to load.");
    const expected = edits.rejectedAttempt ? edits.rejectedAttempt.command.expected_source_version : version;
    if (!expected) throw { data: { source_changed: true, error: "Reload the saved work item before editing." } };
    return {
      expected_source_version: expected,
      mode,
      item_id: data.id,
      project_id:
        (Object.hasOwn(payload, "project_id")
          ? payload.project_id
          : (edits.rejectedAttempt?.command.project_id ?? activeProjectId)) ?? "",
      revision,
      changes: {
        ...edits.rejectedAttempt?.command.changes,
        ...itemEditChanges(payload),
        custom_properties: payload.custom_properties ?? {},
      },
    };
  };
  const handlePublish = async (
    payload: Partial<TIssue>,
    _assetReferences: string[],
    revision?: number,
    draft?: Partial<TIssue>,
    version?: string
  ) => {
    await saveReviewedChange(editCommand(payload, "publish", revision, version), draft);
  };

  const creation = useIssueCreation(
    {
      create: async ({ payload, isDraft: createDraft }) => {
        if (!createDraft && !payload.project_id) throw new Error("Select a project.");
        let response: CreatedWorkItem | undefined;
        if (createDraft) {
          response = await draftMutations.createDraft(payload);
        } else if (requiresProjectIssueCreation(payload, storeType, cycleId, moduleId)) {
          response = await projectActions.createIssue?.(payload.project_id, payload);
        } else if (actions.createIssue) response = await actions.createIssue(payload.project_id, payload);
        if (!response) throw new Error("Work item creation did not complete.");
        return response;
      },
      finalizeAssets: (issue, request) =>
        finalizeAssetUploads(issue.id, issue.project_id, request.isDraft, request.assetReferences ?? []),
      assignCycle: async (issue, { payload, isDraft: createDraft }) => {
        if (!createDraft && payload.cycle_id && payload.cycle_id !== "None") {
          await addIssueToCycle(issue, payload.cycle_id);
        }
      },
      assignModules: async (issue, { payload, isDraft: createDraft }) => {
        if (!createDraft && payload.module_ids?.length) {
          await addIssueToModule(issue, payload.module_ids);
        }
      },
      onSaved: async (issue) => {
        if (creationCurrent() && "sequence_id" in issue) await onSubmit?.(issue);
      },
    },
    creationScope
  );

  const finishCreation = () => {
    if (createMore) issueTitleRef.current?.focus();
    else handleClose();
    creation.reset();
    setChangesMade(null);
  };

  const handleCreateIssue = async (
    payload: Partial<TIssue>,
    is_draft_issue: boolean = false,
    assetReferences: string[] = []
  ): Promise<CreatedWorkItem> => {
    if (!is_draft_issue && !payload.project_id) throw new Error("Select a project.");

    const scope = currentParentCatalogueScope(client, workspaceSlug, payload.project_id ?? undefined).project;
    const access = captureProjectAccessScope(client, {
      accountId: scope.accountId,
      sessionGeneration: scope.sessionGeneration,
      workspaceSlug,
      membershipRole: scope.workspaceRole,
      membershipRevision: scope.membershipRevision,
      projectId: scope.projectId,
      projectMembershipRole: scope.projectMembershipRole,
      projectRole: scope.projectRole,
    });
    try {
      const response = await creation.submit({ payload, isDraft: is_draft_issue, assetReferences });
      if (!creationCurrent() || !access.isCurrent()) return response;

      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("success"),
        message: `${is_draft_issue ? t("draft_created") : t("issue_created_successfully")} `,
        actionItems: updateActionItems(workspaceSlug, response.id, response.project_id, !is_draft_issue),
      });
      finishCreation();
      return response;
    } catch (error) {
      if (creationCurrent() && access.isCurrent())
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("error"),
          message:
            error instanceof IssueCreationIncompleteError
              ? error.message
              : t(is_draft_issue ? "draft_creation_failed" : "issue_creation_failed"),
        });
      throw error;
    }
  };

  const handleFormSubmit = async (
    payload: Partial<TIssue>,
    is_draft_issue: boolean = false,
    assetReferences: string[] = [],
    revision?: number,
    draft?: Partial<TIssue>,
    version?: string
  ) => {
    const { sourceIssueId: _sourceIssueId, ...values } = payload;
    if (beforeFormSubmit) await beforeFormSubmit();
    if (!data?.id) await handleCreateIssue(values, is_draft_issue, assetReferences);
    else await saveReviewedChange(editCommand(values, isDraft ? "draft" : "edit", revision, version), draft);
  };

  const handleFormChange = (formData: Partial<TIssue> | null) => setChangesMade(formData);

  const handleUpdateUploadedAssetIds = (assetId: string) => setUploadedAssetIds((prev) => [...prev, assetId]);

  if (!canOpenIssueEditor(isDraft, allowedProjectIds, activeProjectId)) return null;

  const activeReview = currentEditReview(sourceAccessAllowed, edits);
  const commonIssueModalProps: IssueFormProps = {
    sourceVersion: edits.rejectedAttempt?.command.expected_source_version ?? sourceVersion,
    onReloadSource: async () => {
      if (edits.isPending || edits.recoveryCommand) throw new Error("Resolve the pending save before reloading.");
      await descriptionQuery.refetch({ throwOnError: true });
      if (!edits.isCurrent()) return;
      edits.acknowledge();
      setChangesMade(null);
      setSourceGeneration((current) => current + 1);
    },
    issueTitleRef: issueTitleRef,
    readOnly: creation.hasCreated || edits.isPending,
    persistedData: modalInitialData(data, savedFields, description, isDraft, cycleId, moduleId),
    data: {
      ...modalInitialData(data, savedFields, description, isDraft, cycleId, moduleId),
      ...edits.rejectedAttempt?.draft,
    },
    onAssetUpload: handleUpdateUploadedAssetIds,
    onClose: handleClose,
    onSubmit: (payload, saveDraft = isDraft, references = [], revision, draft, version) =>
      handleFormSubmit(payload, saveDraft, references, revision, draft, version),
    onPublish: handlePublish,
    projectId: activeProjectId ?? null,
    isCreateMoreToggleEnabled: createMore,
    onCreateMoreToggleChange: handleCreateMoreToggleChange,
    isDraft: isDraft,
    moveToIssue: moveToIssue,
    modalTitle: modalTitle,
    primaryButtonText: primaryButtonText,
    isProjectSelectionDisabled: isProjectSelectionDisabled,
  };

  return (
    <>
      <ModalCore
        isOpen={isOpen}
        position={EModalPosition.TOP}
        width={EModalWidth.XXXXL}
        className="rounded-lg !bg-transparent shadow-none transition-[width] ease-linear"
      >
        <IssueEditorSourceGate
          accessState={accessState}
          retryAccess={retryAccess}
          close={handleClose}
          description={description}
          descriptionQuery={descriptionQuery}
          edits={edits}
          save={saveReviewedChange}
        >
          {withDraftIssueWrapper ? (
            <DraftIssueLayout
              key={sourceGeneration}
              {...commonIssueModalProps}
              changesMade={changesMade}
              onChange={handleFormChange}
            />
          ) : (
            <IssueFormRoot key={sourceGeneration} {...commonIssueModalProps} />
          )}
        </IssueEditorSourceGate>
      </ModalCore>
      {activeReview && (
        <ItemEditConfirmation
          workspaceSlug={workspaceSlug}
          review={activeReview.value}
          command={activeReview.command}
          onDecide={edits.decide}
        />
      )}
    </>
  );
}

function IssueEditorSourceGate({
  accessState,
  retryAccess,
  close,
  description,
  descriptionQuery,
  edits,
  save,
  children,
}: {
  accessState: string;
  retryAccess: () => Promise<void>;
  close: () => void;
  description: string | undefined;
  descriptionQuery: ReturnType<typeof useIssueModalSource>["descriptionQuery"];
  edits: ReturnType<typeof useReviewedItemEdit>;
  save: (command: ItemEditCommand) => Promise<void>;
  children: ReactNode;
}) {
  if (accessState !== "ready") return <IssueAccessStatus state={accessState} retry={retryAccess} close={close} />;
  if (edits.recoveryCommand)
    return (
      <ItemEditRecovery command={edits.recoveryCommand} pending={edits.isPending} onRetry={save} onClose={close} />
    );
  if (description === undefined && descriptionQuery.isError)
    return (
      <div role="alert" className="rounded-lg bg-surface-1 p-5">
        Could not load the work item.{" "}
        <button type="button" onClick={() => void descriptionQuery.refetch()}>
          Retry
        </button>
        <button type="button" onClick={close}>
          Close
        </button>
      </div>
    );
  if (description === undefined) return <output className="rounded-lg bg-surface-1 p-5">Loading work item…</output>;
  return children;
}

function IssueAccessStatus({ state, retry, close }: { state: string; retry: () => Promise<void>; close: () => void }) {
  return (
    <div className="rounded-lg bg-surface-1 p-5">
      {state === "pending" ? "Checking work item access…" : "Work item access could not be verified."}
      {state !== "pending" && (
        <button type="button" onClick={() => void retry()}>
          Retry
        </button>
      )}
      <button type="button" onClick={close}>
        Close
      </button>
    </div>
  );
}

function canOpenIssueEditor(isDraft: boolean, allowedProjectIds: string[], activeProjectId: string | undefined | null) {
  return isDraft || !!(allowedProjectIds.length && activeProjectId);
}

function modalInitialData(
  data: Partial<TIssue> | undefined,
  saved: Partial<TIssue> | undefined,
  description: string | undefined,
  isDraft: boolean,
  cycleId: string | undefined,
  moduleId: string | undefined
): Partial<TIssue> {
  const cycle = isDraft ? null : cycleId;
  const modules = !isDraft && moduleId ? [moduleId] : null;
  return {
    ...data,
    ...saved,
    description_html: description,
    cycle_id: saved?.cycle_id !== undefined ? saved.cycle_id : (data?.cycle_id ?? cycle ?? null),
    module_ids: saved?.module_ids ?? data?.module_ids ?? modules,
  };
}

function updateActionItems(
  workspaceSlug: string,
  issueId: string,
  projectId: string | null | undefined,
  show: boolean
) {
  return show && projectId ? (
    <CreateIssueToastActionItems workspaceSlug={workspaceSlug} projectId={projectId} issueId={issueId} />
  ) : undefined;
}

async function captureEditedIssueLists(
  client: ReturnType<typeof useQueryClient>,
  resolveListOwner: ReturnType<typeof useIssueListMutationOwner>,
  workspaceSlug: string,
  projectIds: string[]
) {
  const captured = await Promise.all(
    projectIds.map(async (projectId) => {
      try {
        const owner = await resolveListOwner(workspaceSlug, projectId);
        return owner ? captureIssueListMutationOwner(client, owner) : undefined;
      } catch (error) {
        console.error("Could not capture the work item list before an accepted edit", error);
        return undefined;
      }
    })
  );
  return captured.filter((owner): owner is NonNullable<typeof owner> => owner !== undefined);
}

function issueModalServiceType(configured: EIssuesStoreType | undefined, current: EIssuesStoreType): EIssueServiceType {
  return configured === EIssuesStoreType.EPIC || current === EIssuesStoreType.EPIC
    ? EIssueServiceType.EPICS
    : EIssueServiceType.ISSUES;
}

function issueModalStoreTypes(
  configured: EIssuesStoreType | undefined,
  current: EIssuesStoreType
): {
  storeType: EIssuesStoreType;
  mutationStoreType: Exclude<EIssuesStoreType, EIssuesStoreType.WORKSPACE_DRAFT>;
} {
  const selected = configured ?? current;
  const storeType = selected === EIssuesStoreType.EPIC ? EIssuesStoreType.PROJECT : selected;
  if (storeType === EIssuesStoreType.WORKSPACE_DRAFT) return { storeType, mutationStoreType: EIssuesStoreType.PROJECT };
  return { storeType, mutationStoreType: storeType };
}

function captureEditReceiptOwner(
  client: ReturnType<typeof useQueryClient>,
  command: ItemEditCommand,
  access: ReturnType<typeof useWorkItemDetailScope>
) {
  if (command.mode !== "edit" || !access.isReady) return undefined;
  return captureIssueDetailReconciliation(client, access.scope, [command.item_id]);
}

function issueDetailProjectId(
  activeProjectId: string | null | undefined,
  dataProjectId: string | null | undefined,
  routerProjectId: string | undefined
) {
  return activeProjectId ?? dataProjectId ?? routerProjectId;
}

function editedIssueProjectIds(dataProjectId: string | null | undefined, commandProjectId: string): string[] {
  return [...new Set([dataProjectId, commandProjectId].filter((projectId): projectId is string => !!projectId))];
}

function requiresProjectIssueCreation(
  payload: Partial<TIssue>,
  storeType: EIssuesStoreType,
  cycleId: string | undefined,
  moduleId: string | undefined
) {
  return (
    (storeType === EIssuesStoreType.CYCLE && payload.cycle_id !== cycleId) ||
    (storeType === EIssuesStoreType.MODULE && (!moduleId || !payload.module_ids?.includes(moduleId)))
  );
}

function currentEditReview(accessAllowed: boolean, edits: ReturnType<typeof useReviewedItemEdit>) {
  if (!accessAllowed || !edits.review || !edits.isReviewCurrent()) return undefined;
  return edits.review;
}
