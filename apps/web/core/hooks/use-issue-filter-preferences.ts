/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
import { useParams } from "@tanstack/react-router";
import { useOptionalParentIssuePreferences } from "@/providers/parent-issue-preferences";
import { hashKey } from "@tanstack/react-query";
import { useWorkspaceIssuePreferences } from "@/providers/workspace-issue-preferences";
import type { IIssueFilters, TProfileViews } from "@plane/types";
import { EIssuesStoreType } from "@plane/types";
import { useProjectIssuePreferencesContext } from "@/hooks/use-project-issue-preferences-context";
import { useOptionalProjectViewIssuePreferences } from "@/providers/project-view-issue-preferences";
import type { IProjectIssuePreferencesContext } from "@/hooks/use-project-issue-preferences-context";
import { useLocalIssuePreferences } from "@/providers/local-issue-preferences";

type TIssueDisplayPreferences = Pick<IIssueFilters, "displayFilters" | "displayProperties"> &
  Partial<Pick<IIssueFilters, "richFilters" | "kanbanFilters">>;

/** Keep project-server preferences and non-project store filters at their existing source boundaries. */
export type NativeIssueStoreType = Exclude<EIssuesStoreType, EIssuesStoreType.WORKSPACE_DRAFT>;

type LocalPreferenceReader = ReturnType<typeof useLocalIssuePreferences>;

function localRequestInput(
  storeType: NativeIssueStoreType,
  profileViewId: string | undefined,
  localPreferences: LocalPreferenceReader
) {
  if (storeType === EIssuesStoreType.PROFILE && isProfileView(profileViewId))
    return localPreferences?.profileRequestInput(profileViewId);
  if (storeType === EIssuesStoreType.ARCHIVED) return localPreferences?.archivedRequestInput();
  return undefined;
}

function selectedIssueFilters(
  storeType: NativeIssueStoreType,
  localPreferences: LocalPreferenceReader,
  parentPreferences: ReturnType<typeof useOptionalParentIssuePreferences>,
  projectViewPreferences: ReturnType<typeof useOptionalProjectViewIssuePreferences>,
  selected: ReturnType<typeof selectReadPreferences>
): TIssueDisplayPreferences | undefined {
  if (storeType === EIssuesStoreType.PROFILE || storeType === EIssuesStoreType.ARCHIVED)
    return localPreferences?.issueFilters;
  if (storeType === EIssuesStoreType.CYCLE || storeType === EIssuesStoreType.MODULE) return parentPreferences?.filters;
  if (storeType === EIssuesStoreType.PROJECT_VIEW) return projectViewPreferences?.issueFilters;
  return selected.issueFilters;
}

function selectedReadSignature(
  storeType: NativeIssueStoreType,
  profileViewId: string | undefined,
  localPreferences: LocalPreferenceReader,
  parentPreferences: ReturnType<typeof useOptionalParentIssuePreferences>,
  projectViewPreferences: ReturnType<typeof useOptionalProjectViewIssuePreferences>,
  selected: ReturnType<typeof selectReadPreferences>
) {
  if (storeType === EIssuesStoreType.PROFILE) return localReadSignature(profileViewId, localPreferences);
  if (storeType === EIssuesStoreType.ARCHIVED) return localReadSignature(undefined, localPreferences);
  if (storeType === EIssuesStoreType.CYCLE || storeType === EIssuesStoreType.MODULE)
    return hashKey([parentPreferences?.requestInput?.richFilters, parentPreferences?.requestInput?.displayFilters]);
  if (storeType === EIssuesStoreType.PROJECT_VIEW)
    return hashKey([
      projectViewPreferences?.requestInput.richFilters,
      projectViewPreferences?.requestInput.displayFilters,
    ]);
  return selected.readSignature;
}

function localReadSignature(profileViewId: string | undefined, localPreferences: LocalPreferenceReader) {
  return hashKey([
    localPreferences?.domain,
    localPreferences?.workspaceSlug,
    localPreferences?.entityId,
    localPreferences?.accountId,
    profileViewId,
    localPreferences?.issueFilters.richFilters,
    localPreferences?.issueFilters.displayFilters,
  ]);
}

function selectedRequestInput(
  storeType: NativeIssueStoreType,
  localInput: ReturnType<typeof localRequestInput>,
  projectPreferences: ReturnType<typeof useProjectIssuePreferencesContext>,
  projectViewPreferences: ReturnType<typeof useOptionalProjectViewIssuePreferences>
) {
  if (storeType === EIssuesStoreType.PROJECT || storeType === EIssuesStoreType.EPIC)
    return projectPreferences?.requestInput;
  if (storeType === EIssuesStoreType.PROJECT_VIEW) return projectViewPreferences?.requestInput;
  return localInput;
}

export function useIssueFilterPreferences(storeType: NativeIssueStoreType) {
  const { profileViewId } = useParams({ strict: false });
  const localPreferences = useLocalIssuePreferences();
  const projectPreferences = useProjectIssuePreferencesContext();
  const workspacePreferences = useWorkspaceIssuePreferences();
  const parentPreferences = useOptionalParentIssuePreferences();
  const projectViewPreferences = useOptionalProjectViewIssuePreferences();
  const selected = selectReadPreferences(storeType, projectPreferences, workspacePreferences);
  const localInput = useMemo(
    () => localRequestInput(storeType, profileViewId, localPreferences),
    [localPreferences, profileViewId, storeType]
  );
  const issueFilters = selectedIssueFilters(
    storeType,
    localPreferences,
    parentPreferences,
    projectViewPreferences,
    selected
  );
  const readSignature = selectedReadSignature(
    storeType,
    profileViewId,
    localPreferences,
    parentPreferences,
    projectViewPreferences,
    selected
  );
  return {
    readSignature,
    issueFilters,
    requestInput: selectedRequestInput(storeType, localInput, projectPreferences, projectViewPreferences),
    projectPreferences,
    workspacePreferences,
  };
}

function isProfileView(value: string | undefined): value is TProfileViews {
  return value === "assigned" || value === "created" || value === "subscribed";
}

function selectReadPreferences(
  storeType: NativeIssueStoreType,
  projectPreferences: ReturnType<typeof useProjectIssuePreferencesContext>,
  workspacePreferences: ReturnType<typeof useWorkspaceIssuePreferences>
) {
  const isProject = storeType === EIssuesStoreType.PROJECT || storeType === EIssuesStoreType.EPIC;
  const issueFilters: IProjectIssuePreferencesContext["issueFilters"] | TIssueDisplayPreferences | undefined = isProject
    ? projectPreferences?.issueFilters
    : storeType === EIssuesStoreType.GLOBAL
      ? workspacePreferences?.issueFilters
      : undefined;

  const richFilters = isProject
    ? projectPreferences?.requestInput?.richFilters
    : storeType === EIssuesStoreType.GLOBAL
      ? workspacePreferences?.requestInput?.richFilters
      : undefined;
  return { issueFilters, readSignature: hashKey([richFilters ?? {}, issueFilters?.displayFilters]) };
}
