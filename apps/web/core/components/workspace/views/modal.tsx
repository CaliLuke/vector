/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useRouter } from "@tanstack/react-router";

// plane imports
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IWorkspaceView, TWorkItemFilterExpression } from "@plane/types";

// hooks
import {
  areWorkspaceRichFiltersEquivalent,
  useWorkspaceIssuePreferences,
} from "@/providers/workspace-issue-preferences";
import { useWorkspaceViewActionScope, useWorkspaceViewMutations } from "@/hooks/use-workspace-views";

// local imports
import { ViewForm } from "@/components/views/form";

type Props = {
  data?: IWorkspaceView;
  isOpen: boolean;
  onClose: () => void;
  preLoadedData?: Partial<IWorkspaceView>;
};

function acceptedWorkingExpression(
  working: TWorkItemFilterExpression | undefined,
  payload: Partial<IWorkspaceView>,
  saved: IWorkspaceView
) {
  return (
    working !== undefined &&
    payload.rich_filters !== undefined &&
    areWorkspaceRichFiltersEquivalent(working, payload.rich_filters) &&
    areWorkspaceRichFiltersEquivalent(saved.rich_filters, payload.rich_filters)
  );
}

export function CreateUpdateWorkspaceViewModal(props: Props) {
  const { isOpen, onClose, data, preLoadedData } = props;
  // router
  const router = useRouter();
  const { workspaceSlug } = useParams({ strict: false });
  // store hooks
  const { create, update } = useWorkspaceViewMutations(workspaceSlug);
  const captureActionScope = useWorkspaceViewActionScope(workspaceSlug, isOpen, data?.id);
  const preferences = useWorkspaceIssuePreferences();

  const handleClose = () => {
    onClose();
  };

  const handleCreateView = async (payload: Partial<IWorkspaceView>) => {
    if (!workspaceSlug) return;
    const isCurrent = captureActionScope();
    let saved: IWorkspaceView;
    try {
      saved = await create.mutateAsync({ ...payload, rich_filters: { ...payload.rich_filters } });
    } catch (error) {
      if (isCurrent())
        setToast({ type: TOAST_TYPE.ERROR, title: "Error!", message: "View could not be created. Please try again." });
      throw error;
    }
    if (!isCurrent()) return;
    setToast({ type: TOAST_TYPE.SUCCESS, title: "Success!", message: "View created successfully." });
    void router.navigate({ href: `/${workspaceSlug}/workspace-views/${saved.id}` });
    handleClose();
  };

  const handleUpdateView = async (payload: Partial<IWorkspaceView>) => {
    if (!workspaceSlug || !data) return;
    const submittedRevision = preferences?.revision;
    const workingExpression = preferences?.issueFilters?.richFilters;
    let saved: IWorkspaceView;
    const isCurrent = captureActionScope();
    try {
      saved = await update.mutateAsync({ viewId: data.id, data: { ...payload, query: { ...payload.rich_filters } } });
    } catch (error) {
      if (isCurrent())
        setToast({ type: TOAST_TYPE.ERROR, title: "Error!", message: "View could not be updated. Please try again." });
      throw error;
    }
    if (!isCurrent()) return;
    if (submittedRevision !== undefined && acceptedWorkingExpression(workingExpression, payload, saved))
      preferences?.acknowledgeSave(data.id, submittedRevision, true);
    setToast({ type: TOAST_TYPE.SUCCESS, title: "Success!", message: "View updated successfully." });
    handleClose();
  };

  const handleFormSubmit = async (formData: Partial<IWorkspaceView>) => {
    if (!workspaceSlug) return;

    if (!data) await handleCreateView(formData);
    else await handleUpdateView(formData);
  };

  if (!workspaceSlug) return null;
  if (!isOpen) return null;
  return (
    <ViewForm
      key={data?.id ?? "new"}
      handleFormSubmit={handleFormSubmit}
      handleClose={handleClose}
      data={data}
      preLoadedData={preLoadedData}
      workspaceSlug={workspaceSlug}
    />
  );
}
