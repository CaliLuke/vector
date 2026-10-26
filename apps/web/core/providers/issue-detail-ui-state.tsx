/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useCallback, useContext, useMemo, useState } from "react";
import type { ReactNode } from "react";
import type { TIssue, TIssueLink, TIssueRelationTypes, TIssueServiceType, TWorkItemWidgets } from "@plane/types";
import { EIssueServiceType } from "@plane/types";

export type TPeekIssue = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  nestingLevel?: number;
  isArchived?: boolean;
};

type TIssueRelationModal = {
  issueId: string | null;
  relationType: TIssueRelationTypes | null;
};

type TIssueCrudState = { toggle: boolean; parentIssueId: string | undefined; issue: TIssue | undefined };

type TIssueCrudOperationState = {
  create: TIssueCrudState;
  existing: TIssueCrudState;
};

type IssueDetailUiState = {
  peekIssue: TPeekIssue | undefined;
  relationKey: TIssueRelationTypes | null;
  issueLinkData: TIssueLink | null;
  issueCrudOperationState: TIssueCrudOperationState;
  openWidgets: TWorkItemWidgets[];
  lastWidgetAction: TWorkItemWidgets | null;
  isCreateIssueModalOpen: boolean;
  isIssueLinkModalOpen: boolean;
  isParentIssueModalOpen: string | null;
  isDeleteIssueModalOpen: string | null;
  isArchiveIssueModalOpen: string | null;
  isRelationModalOpen: TIssueRelationModal | null;
  isSubIssuesModalOpen: string | null;
  attachmentDeleteModalId: string | null;
  setPeekIssue: (value: TPeekIssue | undefined) => void;
  setIssueLinkData: (value: TIssueLink | null) => void;
  toggleCreateIssueModal: (value: boolean) => void;
  toggleIssueLinkModal: (value: boolean) => void;
  toggleParentIssueModal: (value: string | null) => void;
  toggleDeleteIssueModal: (value: string | null) => void;
  toggleArchiveIssueModal: (value: string | null) => void;
  toggleRelationModal: (issueId: string | null, relationType: TIssueRelationTypes | null) => void;
  toggleSubIssuesModal: (value: string | null) => void;
  toggleDeleteAttachmentModal: (value: string | null) => void;
  setOpenWidgets: (value: TWorkItemWidgets[]) => void;
  setLastWidgetAction: (value: TWorkItemWidgets) => void;
  toggleOpenWidget: (value: TWorkItemWidgets) => void;
  setRelationKey: (value: TIssueRelationTypes | null) => void;
  setIssueCrudOperationState: (value: TIssueCrudOperationState) => void;
};

const IssueUiContext = createContext<IssueDetailUiState | undefined>(undefined);
const EpicUiContext = createContext<IssueDetailUiState | undefined>(undefined);

const initialCrudState = (): TIssueCrudOperationState => ({
  create: { toggle: false, parentIssueId: undefined, issue: undefined },
  existing: { toggle: false, parentIssueId: undefined, issue: undefined },
});

function useInitialUiState(): IssueDetailUiState {
  const [values, setValues] = useState<
    Omit<
      IssueDetailUiState,
      | "setPeekIssue"
      | "setIssueLinkData"
      | "toggleCreateIssueModal"
      | "toggleIssueLinkModal"
      | "toggleParentIssueModal"
      | "toggleDeleteIssueModal"
      | "toggleArchiveIssueModal"
      | "toggleRelationModal"
      | "toggleSubIssuesModal"
      | "toggleDeleteAttachmentModal"
      | "setOpenWidgets"
      | "setLastWidgetAction"
      | "toggleOpenWidget"
      | "setRelationKey"
      | "setIssueCrudOperationState"
    >
  >(() => ({
    peekIssue: undefined,
    relationKey: null,
    issueLinkData: null,
    issueCrudOperationState: initialCrudState(),
    openWidgets: ["sub-work-items", "links", "attachments"],
    lastWidgetAction: null,
    isCreateIssueModalOpen: false,
    isIssueLinkModalOpen: false,
    isParentIssueModalOpen: null,
    isDeleteIssueModalOpen: null,
    isArchiveIssueModalOpen: null,
    isRelationModalOpen: null,
    isSubIssuesModalOpen: null,
    attachmentDeleteModalId: null,
  }));
  const setPeekIssue = useCallback((value: TPeekIssue | undefined) => {
    setValues((current) => ({ ...current, peekIssue: value }));
  }, []);
  const setIssueLinkData = useCallback((value: TIssueLink | null) => {
    setValues((current) => ({ ...current, issueLinkData: value }));
  }, []);
  const toggleCreateIssueModal = useCallback((value: boolean) => {
    setValues((current) => ({ ...current, isCreateIssueModalOpen: value }));
  }, []);
  const toggleIssueLinkModal = useCallback((value: boolean) => {
    setValues((current) => ({ ...current, isIssueLinkModalOpen: value }));
  }, []);
  const toggleParentIssueModal = useCallback((value: string | null) => {
    setValues((current) => ({ ...current, isParentIssueModalOpen: value }));
  }, []);
  const toggleDeleteIssueModal = useCallback((value: string | null) => {
    setValues((current) => ({ ...current, isDeleteIssueModalOpen: value }));
  }, []);
  const toggleArchiveIssueModal = useCallback((value: string | null) => {
    setValues((current) => ({ ...current, isArchiveIssueModalOpen: value }));
  }, []);
  const toggleRelationModal = useCallback((issueId: string | null, relationType: TIssueRelationTypes | null) => {
    setValues((current) => ({ ...current, isRelationModalOpen: { issueId, relationType } }));
  }, []);
  const toggleSubIssuesModal = useCallback((value: string | null) => {
    setValues((current) => ({ ...current, isSubIssuesModalOpen: value }));
  }, []);
  const toggleDeleteAttachmentModal = useCallback((value: string | null) => {
    setValues((current) => ({ ...current, attachmentDeleteModalId: value }));
  }, []);
  const setOpenWidgets = useCallback((value: TWorkItemWidgets[]) => {
    setValues((current) => ({ ...current, openWidgets: value, lastWidgetAction: null }));
  }, []);
  const setLastWidgetAction = useCallback((value: TWorkItemWidgets) => {
    setValues((current) => ({ ...current, openWidgets: [value], lastWidgetAction: value }));
  }, []);
  const toggleOpenWidget = useCallback((value: TWorkItemWidgets) => {
    setValues((current) => ({
      ...current,
      openWidgets: current.openWidgets.includes(value)
        ? current.openWidgets.filter((widget) => widget !== value)
        : [value, ...current.openWidgets],
    }));
  }, []);
  const setRelationKey = useCallback((value: TIssueRelationTypes | null) => {
    setValues((current) => ({ ...current, relationKey: value }));
  }, []);
  const setIssueCrudOperationState = useCallback((value: TIssueCrudOperationState) => {
    setValues((current) => ({ ...current, issueCrudOperationState: value }));
  }, []);
  const actions = useMemo(
    () => ({
      setPeekIssue,
      setIssueLinkData,
      toggleCreateIssueModal,
      toggleIssueLinkModal,
      toggleParentIssueModal,
      toggleDeleteIssueModal,
      toggleArchiveIssueModal,
      toggleRelationModal,
      toggleSubIssuesModal,
      toggleDeleteAttachmentModal,
      setOpenWidgets,
      setLastWidgetAction,
      toggleOpenWidget,
      setRelationKey,
      setIssueCrudOperationState,
    }),
    [
      setPeekIssue,
      setIssueLinkData,
      toggleCreateIssueModal,
      toggleIssueLinkModal,
      toggleParentIssueModal,
      toggleDeleteIssueModal,
      toggleArchiveIssueModal,
      toggleRelationModal,
      toggleSubIssuesModal,
      toggleDeleteAttachmentModal,
      setOpenWidgets,
      setLastWidgetAction,
      toggleOpenWidget,
      setRelationKey,
      setIssueCrudOperationState,
    ]
  );
  return useMemo(() => ({ ...values, ...actions }), [actions, values]);
}

export function IssueDetailUiStateProvider({ children }: { children: ReactNode }) {
  const issueState = useInitialUiState();
  const epicState = useInitialUiState();
  return (
    <IssueUiContext.Provider value={issueState}>
      <EpicUiContext.Provider value={epicState}>{children}</EpicUiContext.Provider>
    </IssueUiContext.Provider>
  );
}

function useUiState(serviceType: TIssueServiceType) {
  const value = useContext(serviceType === EIssueServiceType.EPICS ? EpicUiContext : IssueUiContext);
  if (!value) throw new Error("Issue detail UI state must be used within IssueDetailUiStateProvider.");
  return value;
}

export function useWorkItemPeek(serviceType: TIssueServiceType = EIssueServiceType.ISSUES) {
  const state = useUiState(serviceType);
  return {
    peekIssue: state.peekIssue,
    setPeekIssue: state.setPeekIssue,
    getIsIssuePeeked: (issueId: string) => state.peekIssue?.issueId === issueId,
  };
}

/** UI-only state for detail dialogs and widgets; server records and operations live in Query/hooks. */
export function useIssueDetailUiState(serviceType: TIssueServiceType = EIssueServiceType.ISSUES) {
  const state = useUiState(serviceType);
  return {
    ...state,
    isAnyModalOpen: Boolean(
      state.isCreateIssueModalOpen ||
      state.isIssueLinkModalOpen ||
      state.isParentIssueModalOpen ||
      state.isDeleteIssueModalOpen ||
      state.isArchiveIssueModalOpen ||
      state.isRelationModalOpen?.issueId ||
      state.isSubIssuesModalOpen ||
      state.attachmentDeleteModalId
    ),
    isPeekOpen: Boolean(state.peekIssue),
    getIsIssuePeeked: (issueId: string) => state.peekIssue?.issueId === issueId,
  };
}
