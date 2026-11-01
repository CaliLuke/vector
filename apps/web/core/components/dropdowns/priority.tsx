/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { SearchableDropdownTrigger, useSearchableDropdown } from "@/components/dropdowns/searchable-options";
import type { ReactNode } from "react";
import { SignalHigh } from "lucide-react";
import { Combobox } from "@headlessui/react";
import { ISSUE_PRIORITIES } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
// types
import { CheckIcon, PriorityIcon, ChevronDownIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import type { TIssuePriorities } from "@plane/types";
// ui
// helpers
import { cn } from "@plane/utils/common";
// hooks
import { usePlatformOS } from "@/hooks/use-platform-os";
// constants
import { BACKGROUND_BUTTON_VARIANTS, BORDER_BUTTON_VARIANTS, BUTTON_VARIANTS_WITHOUT_TEXT } from "./constants";
// types
import type { TDropdownProps } from "./types";

type Props = TDropdownProps & {
  button?: ReactNode;
  dropdownArrow?: boolean;
  dropdownArrowClassName?: string;
  highlightUrgent?: boolean;
  onChange: (val: TIssuePriorities) => void;
  onClose?: () => void;
  value: TIssuePriorities | undefined | null;
  renderByDefault?: boolean;
};

type PriorityPresentation = "border" | "background" | "transparent";
type PriorityDetails = (typeof ISSUE_PRIORITIES)[number];
type ButtonProps = {
  variant: PriorityPresentation;
  className?: string;
  dropdownArrow: boolean;
  dropdownArrowClassName: string;
  hideIcon: boolean;
  hideText: boolean;
  isActive: boolean;
  highlightUrgent: boolean;
  placeholder: string;
  priority: TIssuePriorities | undefined;
  showTooltip: boolean;
  renderToolTipByDefault: boolean;
};

const BUTTON_CLASSES: Record<PriorityPresentation, string> = {
  border: "flex h-full items-center gap-1.5 rounded-sm border-[0.5px] px-2 py-0.5",
  background: "flex h-full items-center gap-1.5 rounded-sm bg-layer-2 px-2 py-0.5",
  transparent: "flex h-full w-full items-center gap-1.5 rounded-sm px-2 hover:bg-layer-transparent-hover",
};
const BORDER_PRIORITY_CLASSES: Record<TIssuePriorities, string> = {
  urgent: "bg-layer-2 border-priority-urgent px-1",
  high: "bg-layer-2 border-priority-high",
  medium: "bg-layer-2 border-priority-medium",
  low: "bg-layer-2 border-priority-low",
  none: "bg-layer-2 border-strong",
};

function PriorityGlyph({
  priority,
  hideText,
  highlightUrgent,
}: Pick<ButtonProps, "priority" | "hideText" | "highlightUrgent">) {
  if (!priority) return <SignalHigh className="size-3" />;
  return (
    <div
      className={cn({
        "rounded-sm border border-priority-urgent p-0.5": priority === "urgent" && !hideText && highlightUrgent,
      })}
    >
      <PriorityIcon
        priority={priority}
        size={12}
        className={cn("flex-shrink-0", {
          "h-3.5 w-3.5": hideText,
          "translate-x-[0.0625rem]": hideText && priority === "high",
          "translate-x-0.5": hideText && priority === "medium",
          "translate-x-1": hideText && priority === "low",
        })}
      />
    </div>
  );
}

function PriorityButtonLabel({
  priority,
  details,
  variant,
  placeholder,
}: Pick<ButtonProps, "priority" | "variant" | "placeholder"> & { details: PriorityDetails | undefined }) {
  const { t } = useTranslation();
  return (
    <span
      className={cn("flex-grow truncate text-body-xs-medium", {
        "text-secondary": priority && priority !== "none",
        "text-placeholder": !priority || priority === "none",
      })}
    >
      {details?.title ?? (variant === "border" ? placeholder : (t("common.priority") ?? placeholder))}
    </span>
  );
}

function PriorityButton(props: ButtonProps) {
  const {
    variant,
    className,
    dropdownArrow,
    dropdownArrowClassName,
    hideIcon,
    hideText,
    isActive,
    highlightUrgent,
    placeholder,
    priority,
    showTooltip,
    renderToolTipByDefault,
  } = props;
  const details = ISSUE_PRIORITIES.find((item) => item.key === priority);
  const { isMobile } = usePlatformOS();
  const { t } = useTranslation();
  const tooltip = variant === "background" ? t(details?.key ?? "none") : (details?.title ?? t("common.none"));
  return (
    <Tooltip
      tooltipHeading={t("priority")}
      tooltipContent={tooltip}
      disabled={!showTooltip}
      isMobile={isMobile}
      renderByDefault={renderToolTipByDefault}
    >
      <div
        className={cn(
          BUTTON_CLASSES[variant],
          variant === "border" && BORDER_PRIORITY_CLASSES[priority ?? "none"],
          {
            "px-0.5": hideText,
            "border-priority-urgent": priority === "urgent" && hideText && highlightUrgent,
            "bg-layer-1": variant === "transparent" && isActive,
          },
          className
        )}
      >
        {!hideIcon && <PriorityGlyph priority={priority} hideText={hideText} highlightUrgent={highlightUrgent} />}
        {!hideText && (
          <PriorityButtonLabel priority={priority} details={details} variant={variant} placeholder={placeholder} />
        )}
        {dropdownArrow && (
          <ChevronDownIcon className={cn("h-2.5 w-2.5 flex-shrink-0", dropdownArrowClassName)} aria-hidden="true" />
        )}
      </div>
    </Tooltip>
  );
}

export function PriorityDropdown(props: Props) {
  //hooks
  const { t } = useTranslation();
  const {
    button,
    buttonClassName,
    buttonContainerClassName,
    buttonVariant,
    className = "",
    disabled = false,
    dropdownArrow = false,
    dropdownArrowClassName = "",
    hideIcon = false,
    highlightUrgent = true,
    onChange,
    onClose,
    placeholder = t("common.priority"),
    placement,
    showTooltip = false,
    tabIndex,
    value = "none",
    renderByDefault = true,
  } = props;

  const dropdown = useSearchableDropdown(placement, onClose);
  const { query, setReferenceElement } = dropdown;
  const options = ISSUE_PRIORITIES.map((priority) => ({
    value: priority.key,
    query: priority.key,
    content: (
      <div className="flex items-center gap-2">
        <PriorityIcon priority={priority.key} size={14} withContainer />
        <span className="flex-grow truncate">{priority.title}</span>
      </div>
    ),
  }));

  const filteredOptions =
    query === "" ? options : options.filter((o) => o.query.toLowerCase().includes(query.toLowerCase()));

  const dropdownOnChange = (val: string | string[] | null) => {
    const priority = ISSUE_PRIORITIES.find((option) => option.key === val);
    if (priority) onChange(priority.key);
  };

  const presentation: PriorityPresentation = BORDER_BUTTON_VARIANTS.includes(buttonVariant)
    ? "border"
    : BACKGROUND_BUTTON_VARIANTS.includes(buttonVariant)
      ? "background"
      : "transparent";

  const comboButton = ({ open }: { open: boolean }) => (
    <SearchableDropdownTrigger
      ref={setReferenceElement}
      customButton={button}
      className={buttonContainerClassName}
      disabled={disabled}
      tabIndex={tabIndex}
    >
      <PriorityButton
        variant={presentation}
        priority={value ?? undefined}
        isActive={open}
        className={buttonClassName}
        highlightUrgent={highlightUrgent}
        dropdownArrow={dropdownArrow && !disabled}
        dropdownArrowClassName={dropdownArrowClassName}
        hideIcon={hideIcon}
        placeholder={placeholder}
        showTooltip={showTooltip}
        hideText={BUTTON_VARIANTS_WITHOUT_TEXT.includes(buttonVariant)}
        renderToolTipByDefault={renderByDefault}
      />
    </SearchableDropdownTrigger>
  );

  return dropdown.render(
    {
      value: value,
      onChange: dropdownOnChange,
      disabled,
      button: comboButton,
      className: cn("h-full", className),
      renderByDefault,
    },
    <>
      {filteredOptions.length > 0 ? (
        filteredOptions.map((option) => (
          <Combobox.Option
            key={option.value}
            value={option.value}
            className={({ active, selected }) =>
              cn(
                `flex w-full cursor-pointer items-center justify-between gap-2 truncate rounded-sm px-1 py-1.5 select-none ${
                  active ? "bg-layer-transparent-hover" : ""
                } ${selected ? "text-primary" : "text-secondary"}`
              )
            }
          >
            {({ selected }) => (
              <>
                <span className="flex-grow truncate">{option.content}</span>
                {selected && <CheckIcon className="h-3.5 w-3.5 flex-shrink-0" />}
              </>
            )}
          </Combobox.Option>
        ))
      ) : (
        <p className="px-1.5 py-1 text-placeholder italic">{t("no_matching_results")}</p>
      )}
    </>,
    t("search")
  );
}
