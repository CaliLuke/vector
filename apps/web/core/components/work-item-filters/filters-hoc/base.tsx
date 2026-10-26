/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, useId } from "react";
import cloneDeep from "lodash-es/cloneDeep.js";
import isEqual from "lodash-es/isEqual.js";
import { useTranslation } from "@plane/i18n";
// plane imports
import type { TSaveViewOptions, TUpdateViewOptions } from "@plane/constants";
import { DEFAULT_FILTER_VISIBILITY_OPTIONS } from "@plane/constants";
import type { TFilterExpression, TFilterValue, TWorkItemFilterExpression, TWorkItemFilterProperty } from "@plane/types";
import { EIssuesStoreType, FILTER_NODE_TYPE } from "@plane/types";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import {
  acceptsFilterExpression,
  addFilterCondition,
  displayFilterConditions,
  filterConditionHasValue,
  filterConditions,
  filterConfigByProperty,
  filterExpressionChanged,
  filterHasActiveValues,
  hasValidFilterValue,
  operatorConfig,
  operatorValueMustReset,
  readFilterExpression,
  removeFilterCondition,
  updateFilterConditionOperator,
  updateFilterConditionProperty,
  updateFilterConditionValue,
  updateFilterGroupOperator,
  writeFilterExpression,
} from "@plane/shared-state";
import { findNodeById } from "@plane/utils/rich-filters";
// plane web imports
import { useWorkItemFiltersConfig } from "@/hooks/work-item-filters/use-work-item-filters-config";
import type { TWorkItemFiltersEntityProps } from "@/hooks/work-item-filters/use-work-item-filters-config";
import { useWorkItemFilterUi } from "@/providers/work-item-filter-ui";
// local imports
import { workItemFiltersAdapter } from "@plane/shared-state";
import type {
  TInitialWorkItemFilters,
  TSharedWorkItemFiltersHOCProps,
  TSharedWorkItemFiltersProps,
  TWorkItemFilterEditor,
} from "./shared";

type TAdditionalWorkItemFiltersProps = {
  saveViewOptions?: TSaveViewOptions<TWorkItemFilterExpression>;
  updateViewOptions?: TUpdateViewOptions<TWorkItemFilterExpression>;
} & TWorkItemFiltersEntityProps;

type TWorkItemFiltersHOCProps = TSharedWorkItemFiltersHOCProps & TAdditionalWorkItemFiltersProps;

// These pages apply authorized property predicates before counts and pagination.
const propertyFilterEntities = new Set([
  EIssuesStoreType.PROJECT,
  EIssuesStoreType.PROJECT_VIEW,
  EIssuesStoreType.GLOBAL,
  EIssuesStoreType.PROFILE,
  EIssuesStoreType.ARCHIVED,
  EIssuesStoreType.MODULE,
  EIssuesStoreType.CYCLE,
]);

type TEditorState = {
  expression: TFilterExpression<TWorkItemFilterProperty> | null;
  initialExpression: TFilterExpression<TWorkItemFilterProperty> | null;
  source: TWorkItemFilterExpression | undefined;
  initialSource: TWorkItemFilterExpression | undefined;
  committedConditionIds: ReadonlySet<string>;
  conversionError: string | null;
  initialConversionFailed: boolean;
  editingError: string | null;
  visible: boolean;
  revision: number;
};

function initialEditorState(expression: TWorkItemFilterExpression | undefined, showOnMount?: boolean): TEditorState {
  const read = readFilterExpression(workItemFiltersAdapter, expression);
  const active = filterHasActiveValues(read.expression, read.committedConditionIds, read.error);
  const visibility = DEFAULT_FILTER_VISIBILITY_OPTIONS;
  const visible = read.error
    ? true
    : visibility.autoSetVisibility
      ? showOnMount === true || active
      : visibility.isVisibleOnMount;
  return {
    expression: read.expression,
    initialExpression: cloneDeep(read.expression),
    source: read.source,
    initialSource: cloneDeep(read.source),
    committedConditionIds: read.committedConditionIds,
    conversionError: read.error,
    initialConversionFailed: Boolean(read.error),
    editingError: null,
    visible: showOnMount ?? visible,
    revision: 0,
  };
}

export const WorkItemFiltersHOC = function WorkItemFiltersHOC(props: TWorkItemFiltersHOCProps) {
  const instanceId = useId();
  const { children, initialWorkItemFilters } = props;
  if (!initialWorkItemFilters)
    return <>{typeof children === "function" ? children({ editor: undefined }) : children}</>;

  const scopeKey = [
    instanceId,
    props.workspaceSlug,
    props.projectId,
    props.entityType,
    props.entityId ?? "temporary",
  ].join(":");
  return <WorkItemFilterRoot key={scopeKey} {...props} initialWorkItemFilters={initialWorkItemFilters} />;
};

type TWorkItemFilterProps = TSharedWorkItemFiltersProps &
  TAdditionalWorkItemFiltersProps & {
    initialWorkItemFilters: TInitialWorkItemFilters;
    children: React.ReactNode | ((props: { editor: TWorkItemFilterEditor }) => React.ReactNode);
  };

const WorkItemFilterRoot = function WorkItemFilterRoot(props: TWorkItemFilterProps) {
  const {
    children,
    entityType,
    filtersToShowByLayout,
    initialWorkItemFilters,
    isTemporary,
    saveViewOptions,
    updateFilters,
    updateViewOptions,
    showOnMount,
    ...entityConfigProps
  } = props;
  const incomingExpression = initialWorkItemFilters.richFilters;
  const [state, setState] = useState(() => initialEditorState(incomingExpression, showOnMount));
  const stateRef = useRef(state);
  useLayoutEffect(() => {
    stateRef.current = state;
  }, [state]);
  const lastIncomingExpression = useRef(incomingExpression);
  const { t } = useTranslation();
  const { state: filterUi, update: updateFilterUi } = useWorkItemFilterUi();
  const isRouteEditor = !isTemporary;

  const workItemFiltersConfig = useWorkItemFiltersConfig({
    allowedFilters: filtersToShowByLayout ?? [],
    propertyFiltersSupported: propertyFilterEntities.has(entityType),
    ...entityConfigProps,
  });
  const conditions = useMemo(() => filterConditions(state.expression), [state.expression]);
  const displayConditions = useMemo(() => displayFilterConditions(state.expression), [state.expression]);
  const status = getEditorStatus(
    state,
    displayConditions,
    isRouteEditor,
    filterUi.visible,
    saveViewOptions,
    updateViewOptions
  );
  const { isDirty, hasActiveFilters, canClear, canSave, canUpdate } = status;

  const notifyExpressionChange = useCallback(
    (expression: TWorkItemFilterExpression) => {
      const reportFailure = (error: unknown) => {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("common.error.label"),
          message: error instanceof Error ? error.message : t("common.something_went_wrong"),
        });
      };
      try {
        void Promise.resolve(updateFilters(expression)).catch(reportFailure);
      } catch (error) {
        reportFailure(error);
      }
    },
    [t, updateFilters]
  );

  useEffect(() => {
    if (isEqual(lastIncomingExpression.current, incomingExpression)) return;
    lastIncomingExpression.current = cloneDeep(incomingExpression);
    const next = initialEditorState(incomingExpression, showOnMount);
    next.visible = stateRef.current.visible;
    next.revision = stateRef.current.revision + 1;
    if (isDirty) {
      next.initialExpression = stateRef.current.initialExpression;
      next.initialSource = stateRef.current.initialSource;
      next.initialConversionFailed = stateRef.current.initialConversionFailed;
    }
    setState(next);
    stateRef.current = next;
  }, [incomingExpression, isDirty, showOnMount]);

  const acceptExpression = useFilterExpressionAcceptance(stateRef, setState, notifyExpressionChange);

  const onToggleVisibility = useCallback(
    (visible?: boolean) => {
      const nextVisible = visible ?? !(filterUi.visible ?? stateRef.current.visible);
      setState((current) => ({ ...current, visible: nextVisible }));
      if (isRouteEditor) updateFilterUi({ visible: nextVisible });
    },
    [filterUi.visible, isRouteEditor, updateFilterUi]
  );

  const conditionHandlers = useFilterConditionHandlers(stateRef, acceptExpression, workItemFiltersConfig.configs);
  const {
    onAddCondition,
    onUpdateConditionProperty,
    onUpdateConditionOperator,
    onUpdateConditionValue,
    onRemoveCondition,
    onUpdateGroupOperator,
  } = conditionHandlers;

  const onClear = useCallback(async () => {
    const current = stateRef.current;
    const canClearCurrent = hasActiveFilters || displayFilterConditions(current.expression).length > 0;
    if (!canClearCurrent) return;
    const revision = current.revision;
    const cleared = readFilterExpression(workItemFiltersAdapter, undefined);
    const shouldNotify = hasActiveFilters || current.expression !== null;
    const next: TEditorState = {
      ...current,
      expression: cleared.expression,
      source: cleared.source,
      committedConditionIds: cleared.committedConditionIds,
      conversionError: null,
      editingError: null,
      revision: revision + 1,
    };
    if (stateRef.current.revision !== revision) return;
    setState(next);
    stateRef.current = next;
    if (shouldNotify) notifyExpressionChange({});
  }, [hasActiveFilters, notifyExpressionChange]);

  const clearRef = useRef(onClear);
  useLayoutEffect(() => {
    clearRef.current = onClear;
  }, [onClear]);
  const clearFilters = useCallback(() => clearRef.current(), []);
  useEffect(() => {
    if (!isRouteEditor) return;
    updateFilterUi({
      visible: filterUi.visible ?? state.visible,
      hasActiveFilters,
      hasUnsavedChanges: canUpdate,
      clearFilters,
    });
  }, [filterUi.visible, state.visible, hasActiveFilters, canUpdate, clearFilters, isRouteEditor, updateFilterUi]);

  useEffect(
    () => () => {
      if (!isRouteEditor) return;
      updateFilterUi({
        visible: undefined,
        hasActiveFilters: false,
        hasUnsavedChanges: false,
        clearFilters: undefined,
      });
    },
    [isRouteEditor, updateFilterUi]
  );

  const onSave = useCallback(async () => {
    if (!canSave || !saveViewOptions) return;
    const current = stateRef.current;
    const result = writeFilterExpression(
      workItemFiltersAdapter,
      current.expression,
      current.committedConditionIds,
      current.conversionError
    );
    if (result.ok) await saveViewOptions.onViewSave(result.value);
    else setState({ ...current, conversionError: result.error });
  }, [canSave, saveViewOptions]);

  const onUpdate = useCallback(async () => {
    if (!canUpdate || !updateViewOptions) return;
    const submitted = stateRef.current;
    const expression = cloneDeep(submitted.expression);
    const result = writeFilterExpression(
      workItemFiltersAdapter,
      expression,
      submitted.committedConditionIds,
      submitted.conversionError
    );
    if (!result.ok) {
      setState({ ...submitted, conversionError: result.error });
      return;
    }
    await updateViewOptions.onViewUpdate(result.value);
    if (stateRef.current.revision !== submitted.revision) return;
    const next: TEditorState = {
      ...stateRef.current,
      initialExpression: cloneDeep(expression),
      initialSource: cloneDeep(result.value),
      initialConversionFailed: false,
    };
    setState(next);
    stateRef.current = next;
  }, [canUpdate, updateViewOptions]);

  const editor: TWorkItemFilterEditor = {
    expression: state.expression,
    conditions,
    displayConditions,
    configs: workItemFiltersConfig.configs,
    isConfigReady: workItemFiltersConfig.areAllConfigsInitialized,
    conversionError: state.conversionError,
    editingError: state.editingError,
    hasActiveFilters,
    hasChanges: isDirty,
    isVisible: status.isVisible,
    saveLabel: saveViewOptions?.label,
    updateLabel: updateViewOptions?.label,
    canClear,
    canSave,
    canUpdate,
    onToggleVisibility,
    onAddCondition,
    onUpdateConditionProperty,
    onUpdateConditionOperator,
    onUpdateConditionValue,
    onRemoveCondition,
    onUpdateGroupOperator,
    onClear,
    onSave,
    onUpdate,
  };

  return (
    <>
      {workItemFiltersConfig.catalogError && (
        <div role="alert" className="flex items-center gap-2 px-4 py-2 text-13 text-secondary">
          Filter choices could not be loaded. Existing filters are preserved.
          <button
            type="button"
            className="text-accent-primary"
            onClick={() => void workItemFiltersConfig.retryCatalogs()}
          >
            Retry
          </button>
        </div>
      )}
      {typeof children === "function" ? <WorkItemFilterRender render={children} editor={editor} /> : children}
    </>
  );
};

function WorkItemFilterRender(props: {
  render: (props: { editor: TWorkItemFilterEditor }) => React.ReactNode;
  editor: TWorkItemFilterEditor;
}) {
  return props.render({ editor: props.editor });
}

function getEditorStatus(
  state: TEditorState,
  displayConditions: ReturnType<typeof displayFilterConditions>,
  isRouteEditor: boolean,
  filterUiVisible: boolean | undefined,
  saveViewOptions: TSaveViewOptions<TWorkItemFilterExpression> | undefined,
  updateViewOptions: TUpdateViewOptions<TWorkItemFilterExpression> | undefined
) {
  const isDirty = isFilterEditorDirty(state);
  const hasActiveFilters = filterHasActiveValues(state.expression, state.committedConditionIds, state.conversionError);
  const canClear = hasActiveFilters || displayConditions.length > 0;
  const canSave =
    !state.conversionError && hasActiveFilters && Boolean(saveViewOptions) && !saveViewOptions?.isDisabled;
  const canUpdate =
    !state.conversionError &&
    Boolean(updateViewOptions) &&
    (isDirty || Boolean(updateViewOptions?.hasAdditionalChanges)) &&
    !updateViewOptions?.isDisabled;
  const isVisible =
    Boolean(state.conversionError) || (isRouteEditor ? (filterUiVisible ?? state.visible) : state.visible);
  return { isDirty, hasActiveFilters, canClear, canSave, canUpdate, isVisible };
}

function isFilterEditorDirty(state: TEditorState) {
  return state.initialConversionFailed
    ? !isEqual(state.initialSource, state.source)
    : filterExpressionChanged(state.initialExpression, state.expression, (condition) =>
        filterConditionHasValue(condition, state.committedConditionIds)
      );
}

type TEditorStateRef = { current: TEditorState };
type TEditorStateSetter = React.Dispatch<React.SetStateAction<TEditorState>>;
type TAcceptExpression = (expression: TEditorState["expression"], shouldNotify: boolean) => void;

function useFilterExpressionAcceptance(
  stateRef: TEditorStateRef,
  setState: TEditorStateSetter,
  notifyExpressionChange: (expression: TWorkItemFilterExpression) => void
) {
  return useCallback(
    (nextExpression: TEditorState["expression"], shouldNotify: boolean) => {
      const current = stateRef.current;
      if (current.conversionError) return;
      if (!acceptsFilterExpression(nextExpression)) {
        setState({ ...current, editingError: "Filters can contain up to five nested levels." });
        return;
      }
      const result = writeFilterExpression(
        workItemFiltersAdapter,
        nextExpression,
        current.committedConditionIds,
        current.conversionError
      );
      if (!result.ok) {
        setState({ ...current, conversionError: result.error, revision: current.revision + 1 });
        return;
      }
      const nextState: TEditorState = {
        ...current,
        expression: nextExpression,
        source: result.value,
        conversionError: null,
        editingError: null,
        revision: current.revision + 1,
      };
      setState(nextState);
      stateRef.current = nextState;
      if (shouldNotify) notifyExpressionChange(result.value);
    },
    [notifyExpressionChange, setState, stateRef]
  );
}

function useFilterConditionHandlers(
  stateRef: TEditorStateRef,
  acceptExpression: TAcceptExpression,
  configs: ReturnType<typeof useWorkItemFiltersConfig>["configs"]
) {
  const onAddCondition = useCallback(
    (property: TWorkItemFilterProperty, operator: import("@plane/types").TSupportedOperators, isNegation: boolean) => {
      const current = stateRef.current;
      if (current.conversionError) return;
      acceptExpression(
        addFilterCondition(current.expression, "and", { property, operator, value: undefined }, isNegation),
        false
      );
    },
    [acceptExpression, stateRef]
  );
  const onUpdateConditionProperty = useCallback(
    (
      conditionId: string,
      property: TWorkItemFilterProperty,
      operator: import("@plane/types").TSupportedOperators,
      isNegation: boolean
    ) => {
      const current = stateRef.current;
      if (!current.expression || current.conversionError) return;
      acceptExpression(
        updateFilterConditionProperty(current.expression, conditionId, property, operator, isNegation),
        true
      );
    },
    [acceptExpression, stateRef]
  );
  const onUpdateConditionOperator = useCallback(
    (conditionId: string, operator: import("@plane/types").TSupportedOperators, isNegation: boolean) => {
      const current = stateRef.current;
      if (!current.expression || current.conversionError) return;
      const previous = findNodeById(current.expression, conditionId);
      if (!previous || previous.type !== FILTER_NODE_TYPE.CONDITION) return;
      const config = filterConfigByProperty(configs, previous.property);
      const currentType = config ? operatorConfig(config, previous.operator)?.type : undefined;
      const nextType = config ? operatorConfig(config, operator)?.type : undefined;
      const shouldResetValue = operatorValueMustReset(previous, currentType, nextType, operator);
      const next = updateFilterConditionOperator(
        current.expression,
        conditionId,
        operator,
        isNegation,
        shouldResetValue
      );
      acceptExpression(next, filterConditionHasValue(previous, current.committedConditionIds));
    },
    [acceptExpression, configs, stateRef]
  );
  const onUpdateConditionValue = useCallback(
    (conditionId: string, value: import("@plane/types").SingleOrArray<TFilterValue>) => {
      const current = stateRef.current;
      if (!current.expression || current.conversionError) return;
      const previous = findNodeById(current.expression, conditionId);
      if (!previous || previous.type !== FILTER_NODE_TYPE.CONDITION) return;
      const { expression, changed } = updateFilterConditionValue(current.expression, conditionId, value);
      if (!changed) return;
      const notify = hasValidFilterValue(value) || filterConditionHasValue(previous, current.committedConditionIds);
      acceptExpression(expression, notify);
    },
    [acceptExpression, stateRef]
  );
  const onRemoveCondition = useCallback(
    (conditionId: string) => {
      const current = stateRef.current;
      if (!current.expression || current.conversionError) return;
      const previous = findNodeById(current.expression, conditionId);
      const hadSavedValue =
        previous?.type === FILTER_NODE_TYPE.CONDITION &&
        filterConditionHasValue(previous, current.committedConditionIds);
      const { expression, shouldNotify } = removeFilterCondition(current.expression, conditionId);
      acceptExpression(expression, shouldNotify || hadSavedValue);
    },
    [acceptExpression, stateRef]
  );
  const onUpdateGroupOperator = useCallback(
    (groupId: string, operator: "and" | "or") => {
      const current = stateRef.current;
      if (!current.expression || current.conversionError) return;
      acceptExpression(updateFilterGroupOperator(current.expression, groupId, operator), true);
    },
    [acceptExpression, stateRef]
  );
  return {
    onAddCondition,
    onUpdateConditionProperty,
    onUpdateConditionOperator,
    onUpdateConditionValue,
    onRemoveCondition,
    onUpdateGroupOperator,
  };
}
