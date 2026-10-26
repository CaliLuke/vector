/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { useStore } from "@tanstack/react-form";
import type { Placement } from "@popperjs/core";

import { usePopper } from "react-popper";
import { Component, Loader } from "lucide-react";
import { Combobox } from "@headlessui/react";
import { getRandomLabelColor } from "@plane/constants";
// plane imports
import { useTranslation } from "@plane/i18n";
import { CheckIcon, SearchIcon, LabelPropertyIcon } from "@plane/propel/icons";
import type { IIssueLabel } from "@plane/types";
import { cn } from "@plane/utils/common";
import { ComboDropDown, ComboSearchInput } from "@plane/ui";
// components
import { IssueLabelsList } from "@/components/ui/labels-list";
// hooks
import { usePlatformOS } from "@/hooks/use-platform-os";
import {
  selectLabelCreationState,
  submitLabelCreationForm,
  useLabelCreationForm,
} from "@/components/labels/use-label-creation-form";
import { CreateLabelOption } from "@/components/labels/create-label-option";
import { useLabelPickerElements } from "@/components/labels/use-label-picker-elements";

export type TWorkItemLabelSelectBaseProps = {
  buttonClassName?: string;
  buttonContainerClassName?: string;
  createLabelEnabled?: boolean;
  disabled?: boolean;
  getLabelById: (labelId: string) => IIssueLabel | null;
  label?: React.ReactNode;
  labelIds: string[];
  onChange: (value: string[]) => void;
  placement?: Placement;
  createLabel?: (data: Partial<IIssueLabel>) => Promise<IIssueLabel>;
  creationScope?: string;
  tabIndex?: number;
  value: string[];
};

export const WorkItemLabelSelectBase = function WorkItemLabelSelectBase(props: TWorkItemLabelSelectBaseProps) {
  return <LabelPicker key={props.creationScope} {...props} />;
};

function LabelPicker(props: TWorkItemLabelSelectBaseProps) {
  const {
    buttonClassName,
    buttonContainerClassName,
    createLabelEnabled = false,
    disabled = false,
    getLabelById,
    label: labelContent,
    labelIds,
    onChange,
    placement,
    createLabel,
    tabIndex,
    value,
  } = props;
  const { dropdownRef, inputRef, referenceElement, setReferenceElement, popperElement, setPopperElement } =
    useLabelPickerElements();
  // plane hooks
  const { t } = useTranslation();
  // store hooks
  const { isMobile } = usePlatformOS();
  const form = useLabelCreationForm({
    errorMessage: t("something_went_wrong_please_try_again"),
    onCreate: async (name, isCurrent) => {
      if (!createLabel || !createLabelEnabled) throw new Error("Label creation is unavailable.");
      const existing = labelsList.find((label) => label.name.toLowerCase() === name.toLowerCase());
      const id = existing ? existing.id : (await createLabel({ name, color: getRandomLabelColor() })).id;
      if (isCurrent()) onChange(Array.from(new Set([...value, id])));
    },
  });
  const { query, submitting, error } = useStore(form.store, selectLabelCreationState);
  const setQuery = (name: string) => form.setFieldValue("name", name);
  // popper-js init
  const { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: placement ?? "bottom-start",
  });
  // derived values
  const labelsList = labelIds.map((labelId) => getLabelById(labelId)).filter((label) => !!label);
  const filteredOptions =
    query === "" ? labelsList : labelsList?.filter((l) => l.name.toLowerCase().includes(query.toLowerCase()));

  const onOpen = () => {
    if (!isMobile) inputRef.current?.focus();
  };

  const searchInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    const q = query.trim();
    if (
      q !== "" &&
      e.key === "Enter" &&
      !e.nativeEvent.isComposing &&
      createLabelEnabled &&
      filteredOptions.length === 0 &&
      !submitting
    ) {
      e.preventDefault();
      void submitLabelCreationForm(form);
    }
  };

  const button = (
    <button
      type="button"
      ref={setReferenceElement}
      className={cn("flex h-full cursor-pointer items-center gap-2 text-11", buttonContainerClassName)}
      disabled={disabled}
      tabIndex={tabIndex}
    >
      {labelContent ? (
        labelContent
      ) : value && value.length > 0 ? (
        <span className={cn("flex h-full items-center justify-center gap-2 text-11", buttonClassName)}>
          <IssueLabelsList labels={value.map((v) => labelsList?.find((l) => l.id === v)) ?? []} length={3} showLength />
        </span>
      ) : (
        <div
          className={cn(
            "flex h-full items-center justify-center gap-1 rounded-sm border-[0.5px] border-strong px-2 py-1 text-11 hover:bg-layer-1",
            buttonClassName
          )}
        >
          <LabelPropertyIcon className="h-3 w-3 flex-shrink-0" />
          <span>{t("labels")}</span>
        </div>
      )}
    </button>
  );

  return (
    <ComboDropDown
      as="div"
      ref={dropdownRef}
      value={value}
      onChange={onChange}
      className="relative h-full flex-shrink-0"
      multiple
      disabled={disabled}
      onOpen={onOpen}
      onClose={() => {
        if (!submitting) form.reset();
      }}
      button={button}
      role="group"
    >
      {({ open, restoreButtonFocusOnClose }) =>
        open && (
          <Combobox.Options
            as="div"
            modal={false}
            className="z-10 my-1 w-48 rounded-sm border-[0.5px] border-strong bg-surface-1 px-2 py-2.5 text-11 shadow-raised-200 focus:outline-none"
            ref={setPopperElement}
            style={styles.popper}
            {...attributes.popper}
          >
            <div className="flex items-center gap-1.5 rounded-sm border border-subtle bg-surface-2 px-2">
              <SearchIcon className="h-3.5 w-3.5 text-placeholder" strokeWidth={1.5} />
              <ComboSearchInput
                as="input"
                ref={inputRef}
                className="w-full bg-transparent py-1 text-11 text-secondary placeholder:text-placeholder focus:outline-none"
                onQueryChange={setQuery}
                readOnly={submitting}
                placeholder={t("search")}
                query={query}
                onEscapeClose={restoreButtonFocusOnClose}
                onKeyDown={searchInputKeyDown}
              />
            </div>
            <div className="mt-2 max-h-48 space-y-1 overflow-y-scroll">
              {error && (
                <p role="alert" className="text-danger-primary">
                  {error}
                </p>
              )}
              {labelsList && filteredOptions ? (
                filteredOptions.length > 0 ? (
                  filteredOptions.map((label) => {
                    const children = labelsList?.filter((l) => l.parent === label.id);

                    if (children.length === 0) {
                      if (!label.parent)
                        return (
                          <Combobox.Option
                            key={label.id}
                            className={({ active }) =>
                              `${
                                active ? "bg-layer-1" : ""
                              } group flex w-full cursor-pointer items-center gap-2 truncate rounded-sm px-1 py-1.5 text-secondary select-none`
                            }
                            value={label.id}
                          >
                            {({ selected }) => (
                              <div className="flex w-full justify-between gap-2 rounded-sm">
                                <div className="flex items-center justify-start gap-2 truncate">
                                  <span
                                    className="h-2.5 w-2.5 flex-shrink-0 rounded-full"
                                    style={{
                                      backgroundColor: label.color,
                                    }}
                                  />
                                  <span className="truncate">{label.name}</span>
                                </div>
                                <div className="flex shrink-0 items-center justify-center rounded-sm p-1">
                                  <CheckIcon className={`h-3 w-3 ${selected ? "opacity-100" : "opacity-0"}`} />
                                </div>
                              </div>
                            )}
                          </Combobox.Option>
                        );
                    } else
                      return (
                        <div key={label.id} className="border-y border-subtle">
                          <div className="flex items-center gap-2 truncate p-2 text-primary select-none">
                            <Component className="h-3 w-3" /> {label.name}
                          </div>
                          <div>
                            {children.map((child) => (
                              <Combobox.Option
                                key={child.id}
                                className={({ active }) =>
                                  `${
                                    active ? "bg-layer-1" : ""
                                  } group flex min-w-[14rem] cursor-pointer items-center gap-2 truncate rounded-sm px-1 py-1.5 text-secondary select-none`
                                }
                                value={child.id}
                              >
                                {({ selected }) => (
                                  <div className="flex w-full justify-between gap-2 rounded-sm">
                                    <div className="flex items-center justify-start gap-2">
                                      <span
                                        className="h-2.5 w-2.5 flex-shrink-0 rounded-full"
                                        style={{
                                          backgroundColor: child?.color,
                                        }}
                                      />
                                      <span>{child.name}</span>
                                    </div>
                                    <div className="flex items-center justify-center rounded-sm p-1">
                                      <CheckIcon className={`h-3 w-3 ${selected ? "opacity-100" : "opacity-0"}`} />
                                    </div>
                                  </div>
                                )}
                              </Combobox.Option>
                            ))}
                          </div>
                        </div>
                      );
                  })
                ) : submitting ? (
                  <Loader className="h-3.5 w-3.5 animate-spin" />
                ) : createLabelEnabled ? (
                  <CreateLabelOption
                    query={query}
                    isSubmitting={submitting}
                    emptyLabel={t("label.create.type")}
                    onCreate={() => void submitLabelCreationForm(form)}
                    onKeyDown={(event) => {
                      if (event.key === "Enter" || event.key === " ") event.stopPropagation();
                    }}
                  />
                ) : (
                  <p className="px-1.5 py-1 text-placeholder italic">{t("no_matching_results")}</p>
                )
              ) : (
                <p className="px-1.5 py-1 text-placeholder italic">{t("loading")}</p>
              )}
            </div>
          </Combobox.Options>
        )
      }
    </ComboDropDown>
  );
}
