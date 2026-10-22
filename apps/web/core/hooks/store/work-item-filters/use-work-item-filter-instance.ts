/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useMemo, useState } from "react";
// plane imports
import type { IWorkItemFilterInstance } from "@plane/shared-state";
import type { EIssuesStoreType } from "@plane/types";
// local imports
import { useWorkItemFilters } from "./use-work-item-filters";

type TWorkItemFilterLifecycleOptions = Parameters<ReturnType<typeof useWorkItemFilters>["getOrCreateFilter"]>[0];

type TWorkItemFilterState = {
  entityType: EIssuesStoreType;
  entityId: string | undefined;
  filter: IWorkItemFilterInstance;
};

export const useWorkItemFilterInstance = (
  entityType: EIssuesStoreType,
  entityId: string | undefined
): IWorkItemFilterInstance | undefined => {
  const { getFilter } = useWorkItemFilters();
  return entityId ? getFilter(entityType, entityId) : undefined;
};

export const useWorkItemFilterLifecycle = (options: TWorkItemFilterLifecycleOptions) => {
  const { entityType, entityId, initialExpression, onExpressionChange, expressionOptions, showOnMount } = options;
  const clearFilterOptions = expressionOptions?.clearFilterOptions;
  const saveViewOptions = expressionOptions?.saveViewOptions;
  const updateViewOptions = expressionOptions?.updateViewOptions;
  const { getOrCreateFilter, deleteFilter } = useWorkItemFilters();
  const filterOptions = useMemo(
    () => ({
      entityType,
      entityId,
      initialExpression,
      onExpressionChange,
      expressionOptions: { clearFilterOptions, saveViewOptions, updateViewOptions },
      showOnMount,
    }),
    [
      entityType,
      entityId,
      initialExpression,
      onExpressionChange,
      clearFilterOptions,
      saveViewOptions,
      updateViewOptions,
      showOnMount,
    ]
  );
  const [filterState, setFilterState] = useState<TWorkItemFilterState>();
  const filter =
    filterState?.entityType === entityType && filterState.entityId === entityId ? filterState.filter : undefined;

  useEffect(() => {
    const nextFilter = getOrCreateFilter(filterOptions);
    setFilterState((current) =>
      current?.entityType === entityType && current.entityId === entityId && current.filter === nextFilter
        ? current
        : { entityType, entityId, filter: nextFilter }
    );
  }, [entityType, entityId, filterOptions, getOrCreateFilter]);

  useEffect(
    () => () => {
      deleteFilter(entityType, entityId);
    },
    [deleteFilter, entityType, entityId]
  );

  return filter;
};
