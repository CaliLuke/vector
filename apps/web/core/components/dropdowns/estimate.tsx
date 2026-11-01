/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { EstimateCatalogueBoundary } from "@/components/estimates/catalogue-boundary";
import { useSearchableDropdown } from "@/components/dropdowns/searchable-options";
import type { ReactNode } from "react";

import { DropdownOption } from "./dropdown-option";
// plane imports
import { useTranslation } from "@plane/i18n";
import { EstimatePropertyIcon, ChevronDownIcon } from "@plane/propel/icons";
import { EEstimateSystem } from "@plane/types";
import { convertMinutesToHoursMinutesString } from "@plane/utils/datetime";
import { cn } from "@plane/utils/common";
// hooks
import { useProjectEstimates } from "@/hooks/use-estimates";
// components
import { DropdownTrigger } from "./buttons";
import { BUTTON_VARIANTS_WITH_TEXT } from "./constants";
// types
import type { TDropdownProps } from "./types";

type Props = TDropdownProps & {
  button?: ReactNode;
  dropdownArrow?: boolean;
  dropdownArrowClassName?: string;
  onChange: (val: string | undefined) => void;
  onClose?: () => void;
  projectId: string | null | undefined;
  value: string | undefined | null;
  renderByDefault?: boolean;
};

type DropdownOptions = { value: string | null; query: string; content: React.ReactNode }[];

export const EstimateDropdown = function EstimateDropdown(props: Props) {
  const {
    buttonVariant,
    className = "",
    disabled = false,
    onChange,
    onClose,
    placeholder = "",
    placement,
    projectId,
    tabIndex,
    value,
    renderByDefault = true,
  } = props;
  // i18n
  const { t } = useTranslation();
  const dropdown = useSearchableDropdown(placement, onClose);
  const { query, setReferenceElement } = dropdown;
  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // store hooks
  const {
    activeEstimate: currentActiveEstimate,
    points,
    isPending,
    isError,
    isAccessDenied,
    retry,
  } = useProjectEstimates(workspaceSlug, projectId);
  const currentActiveEstimateId = currentActiveEstimate?.id;
  const estimatePointById = (pointId: string) => points?.find((point) => point.id === pointId);
  const options: DropdownOptions = (points ?? []).flatMap((point) => {
    if (!point.id) return [];
    return [
      {
        value: point.id,
        query: String(point.value),
        content: (
          <div className="flex items-center gap-2">
            <EstimatePropertyIcon className="h-3 w-3 flex-shrink-0" />
            <span className="flex-grow truncate">
              {currentActiveEstimate?.type === EEstimateSystem.TIME
                ? convertMinutesToHoursMinutesString(Number(point.value))
                : point.value}
            </span>
          </div>
        ),
      },
    ];
  });
  options.unshift({
    value: null,
    query: t("project_settings.estimates.no_estimate"),
    content: (
      <div className="flex items-center gap-2">
        <EstimatePropertyIcon className="h-3 w-3 flex-shrink-0" />
        <span className="flex-grow truncate">{t("project_settings.estimates.no_estimate")}</span>
      </div>
    ),
  });

  const filteredOptions =
    query === "" ? options : options.filter((o) => o.query.toLowerCase().includes(query.toLowerCase()));

  const selectedEstimate = value ? estimatePointById(value) : undefined;

  const dropdownOnChange = (val: string | string[] | null) => {
    onChange(typeof val === "string" ? val : undefined);
  };

  const comboButton = ({ open }: { open: boolean }) => (
    <DropdownTrigger
      appearance={props}
      ref={setReferenceElement}
      isActive={open}
      tooltipHeading={t("project_settings.estimates.label")}
      tooltipContent={selectedEstimate ? selectedEstimate.value : placeholder}
    >
      {!props.hideIcon && <EstimatePropertyIcon className="h-3 w-3 flex-shrink-0" />}
      {(selectedEstimate || placeholder) && BUTTON_VARIANTS_WITH_TEXT.includes(buttonVariant) && (
        <span className="truncate">
          {selectedEstimate ? (
            currentActiveEstimate?.type === EEstimateSystem.TIME ? (
              convertMinutesToHoursMinutesString(Number(selectedEstimate.value))
            ) : (
              selectedEstimate.value
            )
          ) : (
            <span className="text-placeholder">{placeholder}</span>
          )}
        </span>
      )}
      {props.dropdownArrow && (
        <ChevronDownIcon className={cn("h-2.5 w-2.5 flex-shrink-0", props.dropdownArrowClassName)} aria-hidden="true" />
      )}
    </DropdownTrigger>
  );

  return dropdown.render(
    {
      value: value ?? null,
      onChange: dropdownOnChange,
      disabled,
      button: comboButton,
      className: cn("h-full w-full", className),
      tabIndex,
      renderByDefault,
    },
    <>
      <EstimateCatalogueBoundary
        state={{
          isError: isError || isAccessDenied,
          isPending,
          retry: async () => {
            await retry();
          },
        }}
        loading={<p>{t("common.loading")}</p>}
      >
        <EstimateOptions enabled={currentActiveEstimateId !== undefined} options={filteredOptions} />
      </EstimateCatalogueBoundary>
    </>,
    t("common.search.placeholder")
  );
};

function EstimateOptions({ enabled, options }: { enabled: boolean; options: DropdownOptions }) {
  const { t } = useTranslation();
  return (
    <>
      {!enabled ? (
        <div
          className={`flex w-full cursor-pointer items-center justify-between gap-2 truncate rounded-sm px-1 py-1.5 text-secondary select-none`}
        >
          {/* NOTE: This condition renders when estimates are not enabled for the project */}
          <div className="flex flex-grow items-center gap-2">
            <EstimatePropertyIcon className="h-3 w-3 flex-shrink-0" />
            <span className="flex-grow truncate">{t("project_settings.estimates.no_estimate")}</span>
          </div>
        </div>
      ) : (
        <>
          {options.length > 0 ? (
            options.map((option) => (
              <DropdownOption key={option.value} value={option.value}>
                {option.content}
              </DropdownOption>
            ))
          ) : (
            <p className="px-1.5 py-1 text-placeholder italic">{t("common.search.no_matching_results")}</p>
          )}
        </>
      )}
    </>
  );
}
