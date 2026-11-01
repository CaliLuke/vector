/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { useState } from "react";

// plane imports
import { useTranslation } from "@plane/i18n";
import type { IModule } from "@plane/types";
import { ComboDropDown } from "@plane/ui";
import type { TComboDropDownRenderProps } from "@plane/ui";
import { cn } from "@plane/utils/common";
// hooks
// local imports
import { DropdownTrigger } from "../buttons";
import { BUTTON_VARIANTS_WITHOUT_TEXT } from "../constants";
import type { TDropdownProps } from "../types";
import { ModuleButtonContent } from "./button-content";
import { ModuleOptions } from "./module-options";

export type TModuleDropdownProps = TDropdownProps & {
  button?: ReactNode;
  dropdownArrow?: boolean;
  dropdownArrowClassName?: string;
  projectId: string | undefined;
  showCount?: boolean;
  onClose?: () => void;
  renderByDefault?: boolean;
  itemClassName?: string;
} & (
    | {
        multiple: false;
        onChange: (val: string | null) => void;
        value: string | null;
      }
    | {
        multiple: true;
        onChange: (val: string[]) => void;
        value: string[] | null;
      }
  );

type TModuleDropdownBaseProps = TModuleDropdownProps & {
  modules: readonly IModule[] | undefined;
  isError: boolean;
  retry: () => Promise<unknown>;
};

export const ModuleDropdownBase = function ModuleDropdownBase(props: TModuleDropdownBaseProps) {
  const {
    buttonVariant,
    className = "",
    disabled = false,
    modules,
    isError,
    retry,
    itemClassName = "",
    multiple,
    onChange,
    onClose,
    placeholder = "",
    placement,
    projectId,
    renderByDefault = true,
    showCount = false,
    showTooltip = false,
    tabIndex,
    value,
  } = props;
  // i18n
  const { t } = useTranslation();
  // states
  // popper-js refs
  const [referenceElement, setReferenceElement] = useState<HTMLButtonElement | null>(null);

  const comboButton = ({ open }: { open: boolean }) => (
    <DropdownTrigger
      appearance={props}
      ref={setReferenceElement}
      isActive={open}
      hoverBackground
      tabIndex={tabIndex}
      tooltipHeading={t("common.module")}
      tooltipContent={
        Array.isArray(value)
          ? value
              .map((moduleId) => modules?.find((module) => module.id === moduleId)?.name)
              .toString()
              .replaceAll(",", ", ")
          : ""
      }
    >
      <ModuleButtonContent
        modules={modules}
        dropdownArrow={props.dropdownArrow ?? false}
        dropdownArrowClassName={props.dropdownArrowClassName ?? ""}
        hideIcon={props.hideIcon ?? false}
        hideText={BUTTON_VARIANTS_WITHOUT_TEXT.includes(buttonVariant)}
        placeholder={placeholder}
        showCount={showCount}
        showTooltip={showTooltip}
        value={value}
        className={itemClassName}
      />
    </DropdownTrigger>
  );

  const children = ({ open, restoreButtonFocusOnClose }: TComboDropDownRenderProps) =>
    open &&
    projectId && (
      <ModuleOptions
        placement={placement}
        referenceElement={referenceElement}
        multiple={multiple}
        modules={modules}
        isError={isError}
        retry={retry}
        value={value}
        restoreButtonFocusOnClose={restoreButtonFocusOnClose}
      />
    );

  const dropdownProps = {
    as: "div" as const,
    className: cn("h-full", className),
    disabled,
    tabIndex,
    button: comboButton,
    onClose,
    renderByDefault,
    children,
  };

  return multiple ? (
    <ComboDropDown<string, true> {...dropdownProps} multiple value={value ?? []} onChange={onChange} />
  ) : (
    <ComboDropDown<string | null> {...dropdownProps} value={value} onChange={onChange} />
  );
};
