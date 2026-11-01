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
import { DropdownTrigger } from "../buttons";
import { BUTTON_VARIANTS_WITH_TEXT } from "../constants";
import type { TDropdownProps } from "../types";

export type ProjectDropdownProps = TDropdownProps & {
  button?: ReactNode;
  currentProjectId?: string;
  dropdownArrow?: boolean;
  dropdownArrowClassName?: string;
  onClose?: () => void;
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

type Props = ProjectDropdownProps & {
  getProjectById: (projectId: string | null | undefined) => Partial<TProject> | undefined;
  projectIds: string[];
};

const renderIcon = (logoProps: TProject["logo_props"], key?: string) => (
  <span key={key} className="grid h-4 w-4 flex-shrink-0 place-items-center">
    <Logo logo={logoProps} size={14} />
  </span>
);

export const ProjectDropdownBase = function ProjectDropdownBase(props: Props) {
  const {
    buttonVariant,
    className = "",
    currentProjectId,
    disabled = false,
    getProjectById,
    multiple,
    onChange,
    onClose,
    placeholder = "Project",
    placement,
    projectIds,
    renderByDefault = true,
    renderCondition,
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
    if (renderCondition && !renderCondition(projectId)) return undefined;
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

  const getDisplayName = (selectedValue: string | string[] | null, fallbackLabel: string = "") => {
    if (Array.isArray(selectedValue)) {
      const firstProject = getProjectById(selectedValue[0]);
      return selectedValue.length
        ? selectedValue.length === 1
          ? firstProject?.name
          : `${selectedValue.length} projects`
        : fallbackLabel;
    } else {
      return selectedValue ? (getProjectById(selectedValue)?.name ?? fallbackLabel) : fallbackLabel;
    }
  };

  const getProjectIcon = (selectedValue: string | string[] | null) => {
    if (Array.isArray(selectedValue)) {
      return (
        <div className="flex items-center gap-0.5">
          {selectedValue.length > 0 ? (
            selectedValue.map((projectId) => {
              const projectDetails = getProjectById(projectId);
              return projectDetails?.logo_props ? renderIcon(projectDetails.logo_props, projectId) : null;
            })
          ) : (
            <ProjectIcon className="size-3 text-tertiary" />
          )}
        </div>
      );
    } else {
      const projectDetails = getProjectById(selectedValue);
      return projectDetails?.logo_props ? renderIcon(projectDetails.logo_props) : null;
    }
  };

  const comboButton = ({ open }: { open: boolean }) => (
    <DropdownTrigger
      appearance={props}
      ref={setReferenceElement}
      isActive={open}
      tooltipHeading="Project"
      tooltipContent={value?.length ? `${value.length} project${value.length !== 1 ? "s" : ""}` : placeholder}
    >
      {!props.hideIcon && getProjectIcon(value)}
      {BUTTON_VARIANTS_WITH_TEXT.includes(buttonVariant) && (
        <span className="max-w-40 truncate">{getDisplayName(value, placeholder)}</span>
      )}
      {props.dropdownArrow && (
        <ChevronDownIcon className={cn("h-2.5 w-2.5 flex-shrink-0", props.dropdownArrowClassName)} aria-hidden="true" />
      )}
    </DropdownTrigger>
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
