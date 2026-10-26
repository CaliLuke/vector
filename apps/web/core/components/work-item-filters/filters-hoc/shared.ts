/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import type { TSaveViewOptions, TUpdateViewOptions } from "@plane/constants";
import type { EIssuesStoreType, IIssueFilters, TWorkItemFilterExpression, TWorkItemFilterProperty } from "@plane/types";
import type { TFilterExpressionEditor } from "@/components/rich-filters/shared";

export type TSharedWorkItemFiltersProps = {
  entityType: EIssuesStoreType; // entity type (project, cycle, workspace, teamspace, etc)
  filtersToShowByLayout: TWorkItemFilterProperty[];
  updateFilters: (updatedFilters: TWorkItemFilterExpression) => void | Promise<void>;
  isTemporary?: boolean;
  showOnMount?: boolean;
} & ({ isTemporary: true; entityId?: string } | { isTemporary?: false; entityId: string }); // entity id (project_id, cycle_id, workspace_id, etc)

export type TSharedWorkItemFiltersHOCProps = TSharedWorkItemFiltersProps & {
  children: React.ReactNode | ((props: { editor: TWorkItemFilterEditor | undefined }) => React.ReactNode);
  initialWorkItemFilters: TInitialWorkItemFilters | undefined;
};

export type TWorkItemFilterEditor = TFilterExpressionEditor<TWorkItemFilterProperty>;

export type TInitialWorkItemFilters = Pick<IIssueFilters, "richFilters" | "displayFilters" | "displayProperties">;

export type TEnableSaveViewProps = {
  enableSaveView?: boolean;
  saveViewOptions?: Omit<TSaveViewOptions<TWorkItemFilterExpression>, "onViewSave">;
};

export type TEnableUpdateViewProps = {
  enableUpdateView?: boolean;
  updateViewOptions?: Omit<TUpdateViewOptions<TWorkItemFilterExpression>, "onViewUpdate">;
};
