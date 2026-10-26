/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useCallback, useState } from "react";
import { ListFilterPlus } from "lucide-react";
import { Transition } from "@headlessui/react";
// plane imports
import { Button } from "@plane/propel/button";
import type { TFilterProperty } from "@plane/types";
import { cn, EHeaderVariant, Header, Loader } from "@plane/ui";
// local imports
import type { TAddFilterButtonProps } from "./add-filters/button";
import { AddFilterButton } from "./add-filters/button";
import { FilterExpressionTree } from "./expression-tree";
import { FilterRepair } from "./filter-repair";
import type { TFilterExpressionEditor } from "./shared";

export type TFiltersRowProps<P extends TFilterProperty> = {
  buttonConfig?: TAddFilterButtonProps<P>["buttonConfig"];
  disabledAllOperations?: boolean;
  editor: TFilterExpressionEditor<P>;
  variant?: "modal" | "header";
  trackerElements?: {
    clearFilter?: string;
    saveView?: string;
    updateView?: string;
  };
};

function FilterRowWithRepair<P extends TFilterProperty>(props: TFiltersRowProps<P>) {
  if (props.editor.conversionError)
    return (
      <FilterRepair
        message={props.editor.conversionError}
        canClear={props.editor.canClear}
        disabled={props.disabledAllOperations}
        onClear={props.editor.onClear}
      />
    );
  return (
    <>
      {props.editor.editingError && (
        <p role="alert" className="text-sm px-4 py-2 text-secondary">
          {props.editor.editingError}
        </p>
      )}
      <ReadyFiltersRow {...props} />
    </>
  );
}

function ReadyFiltersRow<P extends TFilterProperty>(props: TFiltersRowProps<P>) {
  const {
    buttonConfig,
    disabledAllOperations: disabledAllOperationsProp = false,
    editor,
    variant = "header",
    trackerElements,
  } = props;
  const [isUpdating, setIsUpdating] = useState(false);
  const disabledAllOperations = disabledAllOperationsProp || !editor.isConfigReady;
  const hasAnyConditions = editor.displayConditions.length > 0;
  const hasAvailableOperations = !disabledAllOperations && (editor.canClear || editor.canSave || editor.canUpdate);

  const headerButtonConfig: Partial<TAddFilterButtonProps<P>["buttonConfig"]> = { label: null };
  const modalButtonConfig: Partial<TAddFilterButtonProps<P>["buttonConfig"]> = {
    label: !hasAnyConditions ? "Filters" : null,
  };

  const handleUpdate = useCallback(async () => {
    setIsUpdating(true);
    try {
      await editor.onUpdate();
    } catch {
      // The view owner reports the failure; keep the expression dirty for retry.
    } finally {
      setTimeout(() => setIsUpdating(false), 240);
    }
  }, [editor]);

  const leftContent = (
    <>
      <FilterExpressionTree expression={editor.expression} editor={editor} disabled={disabledAllOperations} />
      <AddFilterButton
        editor={editor}
        buttonConfig={{
          label: null,
          ...(variant === "modal" ? modalButtonConfig : headerButtonConfig),
          size: "lg",
          iconConfig: { shouldShowIcon: true, iconComponent: ListFilterPlus },
          ...buttonConfig,
          isDisabled: disabledAllOperations,
        }}
      />
    </>
  );

  const rightContent = (
    <FilterRowActionButtons
      editor={editor}
      disabled={disabledAllOperations}
      isUpdating={isUpdating}
      onUpdate={handleUpdate}
      trackerElements={trackerElements}
    />
  );

  const mainContent = (
    <div className="flex w-full items-start gap-2 rounded-lg bg-layer-1 px-4 py-2">
      <div className="flex w-full flex-wrap items-center gap-2">{leftContent}</div>
      <div
        className={cn("flex items-center gap-2 border-l border-subtle pl-4", {
          "border-l-transparent pl-0": !hasAvailableOperations,
        })}
      >
        {rightContent}
      </div>
    </div>
  );

  const modalVariant = (
    <div className="flex min-h-11 w-full flex-wrap items-center gap-2 rounded-lg bg-layer-1 p-2">{mainContent}</div>
  );
  const headerVariant = (
    <Header variant={EHeaderVariant.TERNARY} className="min-h-11 bg-surface-1 !px-3">
      {mainContent}
    </Header>
  );

  if (!editor.isConfigReady && !hasAnyConditions) {
    return (
      <RowTransition show={editor.isVisible}>
        <Loader>
          <Loader.Item height="44px" width="100%" className={cn({ "rounded-none": variant === "header" })} />
        </Loader>
      </RowTransition>
    );
  }
  return <RowTransition show={editor.isVisible}>{variant === "modal" ? modalVariant : headerVariant}</RowTransition>;
}

const COMMON_OPERATION_BUTTON_CLASSNAME = "py-1";

type TFilterRowActionButtonsProps<P extends TFilterProperty> = {
  disabled: boolean;
  editor: TFilterExpressionEditor<P>;
  isUpdating: boolean;
  onUpdate: () => void;
  trackerElements?: TFiltersRowProps<P>["trackerElements"];
};

function FilterRowActionButtons<P extends TFilterProperty>(props: TFilterRowActionButtonsProps<P>) {
  const { disabled, editor, isUpdating, onUpdate, trackerElements } = props;
  if (disabled) return null;
  return (
    <>
      <ElementTransition show={editor.canClear}>
        <Button
          variant="secondary"
          className={COMMON_OPERATION_BUTTON_CLASSNAME}
          onClick={editor.onClear}
          data-ph-element={trackerElements?.clearFilter}
        >
          {editor.clearLabel ?? "Clear all"}
        </Button>
      </ElementTransition>
      <ElementTransition show={editor.canSave}>
        <Button
          variant="secondary"
          className={COMMON_OPERATION_BUTTON_CLASSNAME}
          onClick={editor.onSave}
          data-ph-element={trackerElements?.saveView}
        >
          {editor.saveLabel ?? "Save view"}
        </Button>
      </ElementTransition>
      <ElementTransition show={editor.canUpdate}>
        <Button
          variant="secondary"
          className={COMMON_OPERATION_BUTTON_CLASSNAME}
          onClick={onUpdate}
          loading={isUpdating}
          disabled={isUpdating}
          data-ph-element={trackerElements?.updateView}
        >
          {isUpdating ? "Confirming" : (editor.updateLabel ?? "Update view")}
        </Button>
      </ElementTransition>
    </>
  );
}

type TElementTransitionProps = { children: React.ReactNode; show: boolean };
function ElementTransition(props: TElementTransitionProps) {
  return (
    <Transition
      show={props.show}
      enter="transition ease-out duration-200"
      enterFrom="opacity-0 scale-95"
      enterTo="opacity-100 scale-100"
      leave="transition ease-in duration-150"
      leaveFrom="opacity-100 scale-100"
      leaveTo="opacity-0 scale-95"
    >
      {props.children}
    </Transition>
  );
}

type TRowTransitionProps = { children: React.ReactNode; show: boolean };
function RowTransition(props: TRowTransitionProps) {
  return (
    <Transition
      show={props.show}
      enter="transition-all duration-150 ease-out"
      enterFrom="opacity-0 -translate-y-1"
      enterTo="opacity-100 translate-y-0"
      leave="transition-all duration-100 ease-in"
      leaveFrom="opacity-100 translate-y-0"
      leaveTo="opacity-0 -translate-y-1"
    >
      {props.children}
    </Transition>
  );
}

export { FilterRowWithRepair as FiltersRow };
