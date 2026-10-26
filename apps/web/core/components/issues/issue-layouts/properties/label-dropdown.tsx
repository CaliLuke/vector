/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useCallback, useMemo } from "react";
import { useStore } from "@tanstack/react-form";
import type { Placement } from "@popperjs/core";

import { usePopper } from "react-popper";
import { Loader } from "lucide-react";
import { Combobox } from "@headlessui/react";
// plane imports
import { getRandomLabelColor } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { CheckIcon, SearchIcon, ChevronDownIcon } from "@plane/propel/icons";
// types
import type { IIssueLabel } from "@plane/types";
// components
import { ComboDropDown, ComboSearchInput } from "@plane/ui";
import { sortBySelectedFirst } from "@plane/utils/array";
// hooks
import { useProjectLabelMutations } from "@/hooks/use-labels";
import { usePlatformOS } from "@/hooks/use-platform-os";
import {
  selectLabelCreationState,
  submitLabelCreationForm,
  useLabelCreationForm,
} from "@/components/labels/use-label-creation-form";
import { LabelOptionContent } from "@/components/labels/label-option-content";
import { CreateLabelOption } from "@/components/labels/create-label-option";
import { useLabelPickerElements } from "@/components/labels/use-label-picker-elements";

export interface ILabelDropdownProps {
  projectId: string | null;
  value: string[];
  onChange: (data: string[]) => void;
  onClose?: () => void;
  disabled?: boolean;
  defaultOptions?: IIssueLabel[];
  hideDropdownArrow?: boolean;
  className?: string;
  buttonClassName?: string;
  optionsClassName?: string;
  placement?: Placement;
  maxRender?: number;
  renderByDefault?: boolean;
  fullWidth?: boolean;
  fullHeight?: boolean;
  label: React.ReactNode;
}

export function LabelDropdown(props: ILabelDropdownProps) {
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const catalogue = useProjectLabelMutations(workspaceSlug, props.projectId ?? undefined);
  return (
    <ProjectLabelDropdown
      key={JSON.stringify([
        catalogue.scope.accountId,
        catalogue.scope.sessionGeneration,
        workspaceSlug,
        props.projectId,
      ])}
      {...props}
      catalogue={catalogue}
    />
  );
}

function preventLabelDropdownPropagation(event: React.MouseEvent<HTMLDivElement>) {
  event.stopPropagation();
  event.preventDefault();
}

function ProjectLabelDropdown(props: ILabelDropdownProps & { catalogue: ReturnType<typeof useProjectLabelMutations> }) {
  const {
    catalogue,
    projectId,
    value,
    onChange,
    onClose,
    disabled,
    defaultOptions = [],
    hideDropdownArrow = false,
    className,
    buttonClassName = "",
    optionsClassName = "",
    placement,
    maxRender = 2,
    renderByDefault = true,
    fullWidth = false,
    fullHeight = false,
    label,
  } = props;
  const { t } = useTranslation();

  const { dropdownRef, inputRef, referenceElement, setReferenceElement, popperElement, setPopperElement } =
    useLabelPickerElements();

  //hooks
  const { labels: storeLabels, createLabel, canCreate: canCreateLabel, isLoading } = catalogue;
  const form = useLabelCreationForm({
    errorMessage: t("something_went_wrong_please_try_again"),
    onCreate: async (name, isCurrent) => {
      if (!projectId || !canCreateLabel) throw new Error("Label creation is unavailable.");
      const created = await createLabel({ name, color: getRandomLabelColor() });
      if (isCurrent()) onChange(Array.from(new Set([...value, created.id])));
    },
  });
  const { query, submitting, error } = useStore(form.store, selectLabelCreationState);
  const setQuery = (name: string) => form.setFieldValue("name", name);
  const { isMobile } = usePlatformOS();

  let projectLabels: IIssueLabel[] = defaultOptions;
  if (storeLabels && storeLabels.length > 0) projectLabels = storeLabels;

  const options = useMemo(
    () =>
      projectLabels.map((projectLabel) => ({
        value: projectLabel?.id,
        query: projectLabel?.name,
        content: <LabelOptionContent label={projectLabel} />,
      })),
    [projectLabels]
  );

  const filteredOptions = useMemo(
    () =>
      sortBySelectedFirst(
        query === "" ? options : options?.filter((option) => option.query.toLowerCase().includes(query.toLowerCase())),
        value
      ),
    [options, query, value]
  );

  const { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: placement ?? "bottom-start",
    modifiers: [
      {
        name: "preventOverflow",
        options: {
          padding: 12,
        },
      },
    ],
  });

  const onOpen = useCallback(() => form.setErrorMap({ onSubmit: undefined }), [form]);

  const handleClose = () => {
    if (!submitting) form.reset();
    onClose?.();
  };

  const searchInputKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (
      query.trim() &&
      event.key === "Enter" &&
      !event.nativeEvent.isComposing &&
      canCreateLabel &&
      !filteredOptions?.length
    ) {
      event.stopPropagation();
      event.preventDefault();
      void submitLabelCreationForm(form);
    }
  };

  const comboButton = useMemo(
    () => (
      <button
        ref={setReferenceElement}
        type="button"
        className={`clickable flex h-full w-full items-center justify-center gap-1 text-caption-sm-regular ${fullWidth && "hover:bg-layer-1"} ${
          disabled
            ? "cursor-not-allowed text-secondary"
            : value.length <= maxRender
              ? "cursor-pointer"
              : "cursor-pointer hover:bg-layer-1"
        } ${buttonClassName}`}
        disabled={disabled}
      >
        {label}
        {!hideDropdownArrow && !disabled && <ChevronDownIcon className="h-3 w-3" aria-hidden="true" />}
      </button>
    ),
    [buttonClassName, disabled, fullWidth, hideDropdownArrow, label, maxRender, value.length, setReferenceElement]
  );

  return (
    <div className={`${fullHeight ? "h-full" : "h-5"}`} onClick={preventLabelDropdownPropagation}>
      <ComboDropDown
        as="div"
        ref={dropdownRef}
        className={`h-full w-auto max-w-full flex-shrink-0 text-left ${className}`}
        value={value}
        onChange={onChange}
        disabled={disabled}
        onClose={handleClose}
        onOpen={() => {
          onOpen();
          if (!isMobile) inputRef.current?.focus();
        }}
        button={comboButton}
        renderByDefault={renderByDefault}
        multiple
      >
        {({ open, restoreButtonFocusOnClose }) =>
          open && (
            <Combobox.Options modal={false} className="fixed z-10">
              <div
                className={`z-10 my-1 h-auto w-48 rounded-sm border border-strong bg-surface-1 px-2 py-2.5 text-caption-sm-regular whitespace-nowrap shadow-raised-200 focus:outline-none ${optionsClassName}`}
                ref={setPopperElement}
                style={styles.popper}
                {...attributes.popper}
              >
                <div className="flex w-full items-center justify-start rounded-sm border border-subtle bg-surface-2 px-2">
                  <SearchIcon className="h-3.5 w-3.5 text-tertiary" />
                  <ComboSearchInput
                    ref={inputRef}
                    className="w-full bg-transparent px-2 py-1 text-caption-sm-regular text-secondary placeholder:text-placeholder focus:outline-none"
                    query={query}
                    onQueryChange={setQuery}
                    readOnly={submitting}
                    placeholder={t("common.search.label")}
                    onEscapeClose={restoreButtonFocusOnClose}
                    onKeyDown={searchInputKeyDown}
                  />
                </div>
                <div className={`mt-2 max-h-48 space-y-1 overflow-y-scroll`}>
                  {error && (
                    <p role="alert" className="text-danger-primary">
                      {error}
                    </p>
                  )}
                  {isLoading ? (
                    <p className="text-center text-secondary">{t("common.loading")}</p>
                  ) : filteredOptions && filteredOptions.length > 0 ? (
                    filteredOptions.map((option) => (
                      <Combobox.Option
                        key={option.value}
                        value={option.value}
                        onKeyDown={(e) => {
                          if (e.key === "Enter") {
                            e.preventDefault();
                            e.stopPropagation();
                          }
                        }}
                        className={({ active, selected }) =>
                          `flex cursor-pointer items-center justify-between gap-2 truncate rounded-sm px-1 py-1.5 select-none hover:bg-layer-1 ${
                            active ? "bg-layer-1" : ""
                          } ${selected ? "text-primary" : "text-secondary"}`
                        }
                      >
                        {({ selected }) => (
                          <>
                            {option.content}
                            {selected && (
                              <div className="flex-shrink-0">
                                <CheckIcon className={`h-3.5 w-3.5`} />
                              </div>
                            )}
                          </>
                        )}
                      </Combobox.Option>
                    ))
                  ) : submitting ? (
                    <Loader className="h-3.5 w-3.5 animate-spin" />
                  ) : canCreateLabel ? (
                    <CreateLabelOption
                      query={query}
                      isSubmitting={submitting}
                      emptyLabel={t("label.create.type")}
                      onCreate={() => void submitLabelCreationForm(form)}
                    />
                  ) : (
                    <p className="text-left text-secondary">{t("common.search.no_matching_results")}</p>
                  )}
                </div>
              </div>
            </Combobox.Options>
          )
        }
      </ComboDropDown>
    </div>
  );
}
