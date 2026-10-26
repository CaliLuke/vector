import { useCallback, useMemo, useState } from "react";
import cloneDeep from "lodash-es/cloneDeep.js";
import isEqual from "lodash-es/isEqual.js";
import { EViewAccess } from "@plane/types";
import type { IProjectView, TWorkItemFilterExpression } from "@plane/types";
import type { TSharedWorkItemFiltersHOCProps, TEnableSaveViewProps, TEnableUpdateViewProps } from "./shared";
import { removeNillKeys } from "@/components/issues/issue-layouts/utils";

export type ViewFilterPayload = Partial<Pick<IProjectView, "rich_filters" | "display_filters" | "display_properties">>;
type ViewDetails = Pick<IProjectView, "name" | "description" | "access"> &
  Partial<Pick<IProjectView, "logo_props">> &
  ViewFilterPayload;
type ViewDraft = ViewDetails & ViewFilterPayload;

export function useViewFilterActions(
  props: Pick<TSharedWorkItemFiltersHOCProps, "initialWorkItemFilters"> & TEnableSaveViewProps & TEnableUpdateViewProps,
  view: ViewDetails | null | undefined,
  actions: {
    canCreate: boolean | undefined;
    canUpdate: boolean | undefined;
    update: (payload: ViewFilterPayload) => Promise<void>;
  }
) {
  const filters = props.initialWorkItemFilters;
  const createLabel = props.saveViewOptions?.label;
  const updateLabel = props.updateViewOptions?.label;
  const { canCreate, canUpdate, update } = actions;
  const [createViewPayload, setPayload] = useState<ViewDraft>();
  const [isCreateViewModalOpen, setOpen] = useState(false);
  const filterPayload = useCallback(
    (expression: TWorkItemFilterExpression): ViewFilterPayload => ({
      rich_filters: cloneDeep(expression),
      display_filters: cloneDeep(filters?.displayFilters),
      display_properties: cloneDeep(filters?.displayProperties),
    }),
    [filters]
  );
  const save = useCallback(
    (expression: TWorkItemFilterExpression) => {
      setPayload({
        name: view ? `${view.name} 2` : "Untitled",
        description: view?.description ?? "",
        logo_props: view?.logo_props,
        access: view?.access ?? EViewAccess.PUBLIC,
        ...filterPayload(expression),
      });
      setOpen(true);
    },
    [view, filterPayload]
  );
  const saveViewOptions = useMemo(
    () => ({ label: createLabel, isDisabled: !canCreate, onViewSave: save }),
    [createLabel, canCreate, save]
  );
  const hasAdditionalChanges =
    !isEqual(filters?.displayFilters, view?.display_filters) ||
    !isEqual(removeNillKeys(filters?.displayProperties), removeNillKeys(view?.display_properties));
  const updateViewOptions = useMemo(
    () => ({
      label: updateLabel,
      isDisabled: !canUpdate,
      hasAdditionalChanges,
      onViewUpdate: (expression: TWorkItemFilterExpression) => update(filterPayload(expression)),
    }),
    [updateLabel, canUpdate, hasAdditionalChanges, update, filterPayload]
  );
  const close = useCallback(() => {
    setOpen(false);
    setPayload(undefined);
  }, []);
  return { createViewPayload, isCreateViewModalOpen, saveViewOptions, updateViewOptions, close };
}
