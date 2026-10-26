/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { DropdownOption } from "@/components/dropdowns/dropdown-option";

import type { ReactNode } from "react";
import { useRef } from "react";

// plane imports
import { useTranslation } from "@plane/i18n";
import { Logo } from "@plane/propel/emoji-icon-picker";
import { ProjectIcon, ChevronDownIcon } from "@plane/propel/icons";
import { ComboDropDown } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { sortBySelectedFirst } from "@plane/utils/array";
// components
// hooks
import { useSearchableDropdown } from "../searchable-options";
// plane web imports
import type { TProject } from "@plane/types";
// local imports
import { DropdownButton } from "../buttons";
import { BUTTON_VARIANTS_WITH_TEXT } from "../constants";
import type { TDropdownProps } from "../types";

type Props = TDropdownProps & {
  button?: ReactNode;
  currentProjectId?: string;
  dropdownArrow?: boolean;
  dropdownArrowClassName?: string;
  getProjectById: (projectId: string | null | undefined) => Partial<TProject> | undefined;
  onClose?: () => void;
  projectIds: string[];
  renderByDefault?: boolean;
  renderCondition?: (projectId: string) => boolean;
} & (
    | {
        multiple: false;
        onChange: (val: string) => void;
        value: string | null;
      }
    | {
        multiple: true;
        onChange: (val: string[]) => void;
        value: string[];
      }
  );

export const ProjectDropdownBase = function ProjectDropdownBase(props: Props) {
  const {
    button,
    buttonClassName,
    buttonContainerClassName,
    buttonVariant,
    className = "",
    currentProjectId,
    disabled = false,
    dropdownArrow = false,
    dropdownArrowClassName = "",
    getProjectById,
    hideIcon = false,
    multiple,
    onChange,
    onClose,
    placeholder = "Project",
    placement,
    projectIds,
    renderByDefault = true,
    renderCondition,
    showTooltip = false,
    tabIndex,
    value,
  } = props;
  // refs
  const dropdownRef = useRef<HTMLDivElement | null>(null);
  const { t } = useTranslation();
  const dropdown = useSearchableDropdown(placement, onClose);
  const { query, setReferenceElement } = dropdown;
  // store hooks
  const options = projectIds?.map((projectId) => {
    const projectDetails = getProjectById(projectId);
    if (renderCondition && !renderCondition(projectId)) return;
    return {
      value: projectId,
      query: `${projectDetails?.name}`,
      content: (
        <div className="flex items-center gap-2">
          {projectDetails?.logo_props && (
            <span className="grid h-4 w-4 flex-shrink-0 place-items-center">
              <Logo logo={projectDetails?.logo_props} size={12} />
            </span>
          )}
          <span className="flex-grow truncate">{projectDetails?.name}</span>
        </div>
      ),
    };
  });

  const filteredOptions = sortBySelectedFirst(
    (query === ""
      ? options?.filter((o) => o?.value !== currentProjectId)
      : options?.filter((o) => o?.value !== currentProjectId && o?.query.toLowerCase().includes(query.toLowerCase()))
    )?.filter((o): o is NonNullable<typeof o> => o !== undefined),
    value
  );

  const getDisplayName = (value: string | string[] | null, placeholder: string = "") => {
    if (Array.isArray(value)) {
      const firstProject = getProjectById(value[0]);
      return value.length ? (value.length === 1 ? firstProject?.name : `${value.length} projects`) : placeholder;
    } else {
      return value ? (getProjectById(value)?.name ?? placeholder) : placeholder;
    }
  };

  const getProjectIcon = (value: string | string[] | null) => {
    const renderIcon = (logoProps: TProject["logo_props"]) => (
      <span className="grid h-4 w-4 flex-shrink-0 place-items-center">
        <Logo logo={logoProps} size={14} />
      </span>
    );

    if (Array.isArray(value)) {
      return (
        <div className="flex items-center gap-0.5">
          {value.length > 0 ? (
            value.map((projectId) => {
              const projectDetails = getProjectById(projectId);
              return projectDetails?.logo_props ? renderIcon(projectDetails.logo_props) : null;
            })
          ) : (
            <ProjectIcon className="size-3 text-tertiary" />
          )}
        </div>
      );
    } else {
      const projectDetails = getProjectById(value);
      return projectDetails?.logo_props ? renderIcon(projectDetails.logo_props) : null;
    }
  };

  const comboButton = ({ open }: { open: boolean }) =>
    button ? (
      <button
        ref={setReferenceElement}
        type="button"
        className={cn("clickable block h-full w-full outline-none", buttonContainerClassName)}
        disabled={disabled}
      >
        {button}
      </button>
    ) : (
      <button
        ref={setReferenceElement}
        type="button"
        className={cn(
          "clickable block h-full max-w-full outline-none",
          {
            "cursor-not-allowed text-secondary": disabled,
            "cursor-pointer": !disabled,
          },
          buttonContainerClassName
        )}
        disabled={disabled}
      >
        <DropdownButton
          className={buttonClassName}
          isActive={open}
          tooltipHeading="Project"
          tooltipContent={value?.length ? `${value.length} project${value.length !== 1 ? "s" : ""}` : placeholder}
          showTooltip={showTooltip}
          variant={buttonVariant}
          renderToolTipByDefault={renderByDefault}
        >
          {!hideIcon && getProjectIcon(value)}
          {BUTTON_VARIANTS_WITH_TEXT.includes(buttonVariant) && (
            <span className="max-w-40 truncate">{getDisplayName(value, placeholder)}</span>
          )}
          {dropdownArrow && (
            <ChevronDownIcon className={cn("h-2.5 w-2.5 flex-shrink-0", dropdownArrowClassName)} aria-hidden="true" />
          )}
        </DropdownButton>
      </button>
    );

  const children = dropdown.renderOptions(<ProjectOptions options={filteredOptions} />, t("search"));

  const dropdownProps = {
    ref: dropdownRef,
    tabIndex,
    className: cn("h-full", className),
    disabled,
    onClose: dropdown.onClose,
    onOpen: dropdown.onOpen,
    button: comboButton,
    renderByDefault,
    children,
  };

  return multiple ? (
    <ComboDropDown<string, true> {...dropdownProps} as="div" multiple value={value} onChange={onChange} />
  ) : (
    <ComboDropDown<string | null>
      {...dropdownProps}
      as="div"
      value={value}
      onChange={(selected) => {
        if (selected !== null) onChange(selected);
      }}
    />
  );
};

function ProjectOptions({ options }: { options: { value: string; content: ReactNode }[] | undefined }) {
  const { t } = useTranslation();
  if (!options) return <p className="px-1.5 py-1 text-placeholder italic">{t("loading")}</p>;
  if (!options.length) return <p className="px-1.5 py-1 text-placeholder italic">{t("no_matching_results")}</p>;
  return options.map((option) => (
    <DropdownOption key={option.value} value={option.value}>
      {option.content}
    </DropdownOption>
  ));
}
