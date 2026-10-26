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
import { DropdownButton } from "../buttons";
import { BUTTON_VARIANTS_WITHOUT_TEXT } from "../constants";
import type { TDropdownProps } from "../types";
import { ModuleButtonContent } from "./button-content";
import { ModuleOptions } from "./module-options";

type TModuleDropdownBaseProps = TDropdownProps & {
  button?: ReactNode;
  dropdownArrow?: boolean;
  dropdownArrowClassName?: string;
  modules: readonly IModule[] | undefined;
  isError: boolean;
  retry: () => Promise<unknown>;
  itemClassName?: string;
  onClose?: () => void;
  projectId: string | undefined;
  renderByDefault?: boolean;
  showCount?: boolean;
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

export const ModuleDropdownBase = function ModuleDropdownBase(props: TModuleDropdownBaseProps) {
  const {
    button,
    buttonClassName,
    buttonContainerClassName,
    buttonVariant,
    className = "",
    disabled = false,
    dropdownArrow = false,
    dropdownArrowClassName = "",
    modules,
    isError,
    retry,
    hideIcon = false,
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

  const comboButton = ({ open }: { open: boolean }) =>
    button ? (
      <button
        ref={setReferenceElement}
        type="button"
        className={cn("clickable block h-full w-full outline-none hover:bg-layer-1", buttonContainerClassName)}
        disabled={disabled}
        tabIndex={tabIndex}
      >
        {button}
      </button>
    ) : (
      <button
        ref={setReferenceElement}
        type="button"
        className={cn(
          "clickable block h-full max-w-full outline-none hover:bg-layer-1",
          {
            "cursor-not-allowed text-secondary": disabled,
            "cursor-pointer": !disabled,
          },
          buttonContainerClassName
        )}
        disabled={disabled}
        tabIndex={tabIndex}
      >
        <DropdownButton
          className={buttonClassName}
          isActive={open}
          tooltipHeading={t("common.module")}
          tooltipContent={
            Array.isArray(value)
              ? `${value
                  .map((moduleId) => modules?.find((module) => module.id === moduleId)?.name)
                  .toString()
                  .replaceAll(",", ", ")}`
              : ""
          }
          showTooltip={showTooltip}
          variant={buttonVariant}
          renderToolTipByDefault={renderByDefault}
        >
          <ModuleButtonContent
            modules={modules}
            dropdownArrow={dropdownArrow}
            dropdownArrowClassName={dropdownArrowClassName}
            hideIcon={hideIcon}
            hideText={BUTTON_VARIANTS_WITHOUT_TEXT.includes(buttonVariant)}
            placeholder={placeholder}
            showCount={showCount}
            showTooltip={showTooltip}
            value={value}
            className={itemClassName}
          />
        </DropdownButton>
      </button>
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
