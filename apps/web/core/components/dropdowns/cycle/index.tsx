/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { useState } from "react";

import { useParams } from "@tanstack/react-router";
import { useTranslation } from "@plane/i18n";
// ui
import { CycleIcon, ChevronDownIcon } from "@plane/propel/icons";
import { ComboDropDown } from "@plane/ui";
// helpers
import { cn } from "@plane/utils/common";
// hooks
import { useProjectCycles } from "@/hooks/use-cycles";
// local components and constants
import { DropdownTrigger } from "../buttons";
import { BUTTON_VARIANTS_WITH_TEXT } from "../constants";
import type { TDropdownProps } from "../types";
import { CycleOptions } from "./cycle-options";

type Props = TDropdownProps & {
  button?: ReactNode;
  dropdownArrow?: boolean;
  dropdownArrowClassName?: string;
  onChange: (val: string | null) => void;
  onClose?: () => void;
  projectId: string | undefined;
  value: string | null;
  canRemoveCycle?: boolean;
  renderByDefault?: boolean;
  currentCycleId?: string;
};

export const CycleDropdown = function CycleDropdown(props: Props) {
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
    canRemoveCycle = true,
    renderByDefault = true,
    currentCycleId,
  } = props;
  // i18n
  const { t } = useTranslation();
  // states

  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const catalogue = useProjectCycles(workspaceSlug, projectId);
  // popper-js refs
  const [referenceElement, setReferenceElement] = useState<HTMLButtonElement | null>(null);

  const selectedName = catalogue.cycles?.find((cycle) => cycle.id === value)?.name ?? null;

  const dropdownOnChange = (val: string | string[] | null) => {
    if (!Array.isArray(val)) onChange(val);
  };

  const comboButton = ({ open }: { open: boolean }) => (
    <DropdownTrigger
      appearance={props}
      ref={setReferenceElement}
      isActive={open}
      hoverBackground
      tabIndex={tabIndex}
      tooltipHeading={t("common.cycle")}
      tooltipContent={selectedName ?? placeholder}
    >
      {!props.hideIcon && <CycleIcon className="h-3 w-3 flex-shrink-0" />}
      {BUTTON_VARIANTS_WITH_TEXT.includes(buttonVariant) && (!!selectedName || !!placeholder) && (
        <span className="max-w-40 truncate">{selectedName ?? placeholder}</span>
      )}
      {props.dropdownArrow && (
        <ChevronDownIcon className={cn("h-2.5 w-2.5 flex-shrink-0", props.dropdownArrowClassName)} aria-hidden="true" />
      )}
    </DropdownTrigger>
  );

  return (
    <ComboDropDown<string | null>
      as="div"
      className={cn("h-full", className)}
      value={value}
      onChange={dropdownOnChange}
      disabled={disabled}
      button={comboButton}
      onClose={onClose}
      renderByDefault={renderByDefault}
    >
      {({ open, restoreButtonFocusOnClose }: { open: boolean; restoreButtonFocusOnClose: () => void }) =>
        open &&
        projectId && (
          <CycleOptions
            cycles={catalogue.cycles}
            isError={catalogue.isError || catalogue.isAccessDenied}
            retry={catalogue.retry}
            placement={placement}
            referenceElement={referenceElement}
            canRemoveCycle={canRemoveCycle}
            currentCycleId={currentCycleId}
            restoreButtonFocusOnClose={restoreButtonFocusOnClose}
          />
        )
      }
    </ComboDropDown>
  );
};
