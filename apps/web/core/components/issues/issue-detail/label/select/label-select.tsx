/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Fragment, useState } from "react";
import { useStore } from "@tanstack/react-form";
import { usePopper } from "react-popper";
import { Loader } from "lucide-react";
import { Combobox } from "@headlessui/react";
// plane imports
import { getRandomLabelColor } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { CheckIcon, SearchIcon, PlusIcon } from "@plane/propel/icons";
import type { IIssueLabel } from "@plane/types";
// helpers
import { getTabIndex } from "@plane/utils/tab-indices";
// hooks
import { useProjectLabels } from "@/hooks/use-labels";
import { usePlatformOS } from "@/hooks/use-platform-os";
import {
  selectLabelCreationState,
  submitLabelCreationForm,
  useLabelCreationForm,
} from "@/components/labels/use-label-creation-form";
import { LabelOptionContent } from "@/components/labels/label-option-content";
//constants
export interface IIssueLabelSelect {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  values: string[];
  onSelect: (_labelIds: string[]) => void;
  onAddLabel: (data: Partial<IIssueLabel>) => Promise<IIssueLabel>;
}

export function IssueLabelSelect(props: IIssueLabelSelect) {
  const catalogue = useProjectLabels(props.workspaceSlug, props.projectId);
  return (
    <DetailLabelPicker
      key={JSON.stringify([
        catalogue.scope.accountId,
        catalogue.scope.sessionGeneration,
        props.workspaceSlug,
        props.projectId,
        props.issueId,
      ])}
      {...props}
      catalogue={catalogue}
    />
  );
}

function DetailLabelPicker(props: IIssueLabelSelect & { catalogue: ReturnType<typeof useProjectLabels> }) {
  const { catalogue, issueId, values, onSelect, onAddLabel } = props;
  const { t } = useTranslation();
  // store hooks
  const { isMobile } = usePlatformOS();
  const { labels: projectLabels, isLoading, canCreate: canCreateLabel } = catalogue;
  // states
  const [referenceElement, setReferenceElement] = useState<HTMLButtonElement | null>(null);
  const [popperElement, setPopperElement] = useState<HTMLElement | null>(null);
  const form = useLabelCreationForm({
    errorMessage: t("something_went_wrong_please_try_again"),
    onCreate: async (name, isCurrent) => {
      if (!canCreateLabel) throw new Error("Label creation is unavailable.");
      const created = await onAddLabel({ name, color: getRandomLabelColor() });
      if (isCurrent()) onSelect(Array.from(new Set([...values, created.id])));
    },
  });
  const { query, submitting, error } = useStore(form.store, selectLabelCreationState);
  const setQuery = (name: string) => form.setFieldValue("name", name);

  const { baseTabIndex } = getTabIndex(undefined, isMobile);

  const options = (projectLabels ?? []).map((label) => ({
    value: label.id,
    query: label.name,
    content: <LabelOptionContent label={label} />,
  }));

  const filteredOptions =
    query === "" ? options : options?.filter((option) => option.query.toLowerCase().includes(query.toLowerCase()));

  const { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: "bottom-start",
    modifiers: [
      {
        name: "preventOverflow",
        options: {
          padding: 12,
        },
      },
    ],
  });

  const issueLabels = values ?? [];

  const label = <span className="text-body-xs-medium text-placeholder">{t("label.select")}</span>;

  const searchInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (query === "" && e.key === "Escape") {
      // The search input lives inside the popup and unmounts when Headless UI
      // closes it. Move focus to the persistent trigger before that happens.
      referenceElement?.focus();
    }
    if (query !== "" && e.key === "Escape") {
      e.stopPropagation();
      setQuery("");
    }

    if (query.trim() && e.key === "Enter" && !e.nativeEvent.isComposing && canCreateLabel && !filteredOptions.length) {
      e.stopPropagation();
      e.preventDefault();
      void submitLabelCreationForm(form);
    }
  };

  if (!issueId || !values) return <></>;

  return (
    <>
      <Combobox
        as="div"
        className="size-full flex-shrink-0 text-left"
        value={issueLabels}
        onChange={(value) => onSelect(value)}
        multiple
      >
        <Combobox.Button as={Fragment}>
          <Button ref={setReferenceElement} type="button" variant="tertiary" size="sm" prependIcon={<PlusIcon />}>
            {label}
          </Button>
        </Combobox.Button>

        <Combobox.Options
          as="div"
          modal={false}
          className={`z-10 my-1 w-48 rounded-sm border border-strong bg-surface-1 py-2.5 text-11 whitespace-nowrap shadow-raised-200 focus:outline-none`}
          ref={setPopperElement}
          style={styles.popper}
          {...attributes.popper}
        >
          <div className="px-2">
            <div className="flex w-full items-center justify-start rounded-sm border border-subtle bg-surface-2 px-2">
              <SearchIcon className="h-3.5 w-3.5 text-tertiary" />
              <Combobox.Input
                className="w-full bg-transparent px-2 py-1 text-11 text-secondary placeholder:text-placeholder focus:outline-none"
                value={query}
                readOnly={submitting}
                onChange={(e) => setQuery(e.target.value)}
                placeholder={t("common.search.label")}
                displayValue={(assigned: any) => assigned?.name}
                onKeyDown={searchInputKeyDown}
                tabIndex={baseTabIndex}
              />
            </div>
          </div>
          <div className={`vertical-scrollbar mt-2 scrollbar-sm max-h-48 space-y-1 overflow-y-scroll px-2 pr-0`}>
            {error && (
              <p role="alert" className="text-danger-primary">
                {error}
              </p>
            )}
            {isLoading ? (
              <p className="text-center text-secondary">{t("common.loading")}</p>
            ) : filteredOptions.length > 0 ? (
              filteredOptions.map((option) => (
                <Combobox.Option
                  key={option.value}
                  value={option.value}
                  className={({ selected }) =>
                    `flex cursor-pointer items-center justify-between gap-2 truncate rounded-sm px-1 py-1.5 select-none hover:bg-layer-1 ${
                      selected ? "text-primary" : "text-secondary"
                    }`
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
              <Loader className="spin h-3.5 w-3.5" />
            ) : canCreateLabel ? (
              <button
                type="button"
                disabled={!query.trim() || submitting}
                onClick={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  if (!query.length) return;
                  void submitLabelCreationForm(form);
                }}
                className={`text-left text-secondary ${query.length ? "cursor-pointer" : "cursor-default"}`}
              >
                {query.length ? (
                  <>
                    {/* TODO: Translate here */}+ Add <span className="text-primary">&quot;{query}&quot;</span> to
                    labels
                  </>
                ) : (
                  t("label.create.type")
                )}
              </button>
            ) : (
              <p className="text-left text-secondary">{t("common.search.no_matching_results")}</p>
            )}
          </div>
        </Combobox.Options>
      </Combobox>
    </>
  );
}
