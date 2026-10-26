/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { ModuleIcon, ChevronDownIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import { cn } from "@plane/utils/common";
// hooks
import type { IModule } from "@plane/types";
import { usePlatformOS } from "@/hooks/use-platform-os";

type ModuleButtonContentProps = {
  modules: readonly IModule[] | undefined;
  dropdownArrow: boolean;
  dropdownArrowClassName: string;
  hideIcon: boolean;
  hideText: boolean;
  placeholder?: string;
  showCount: boolean;
  showTooltip?: boolean;
  value: string | string[] | null;
  className?: string;
};

export function ModuleButtonContent(props: ModuleButtonContentProps) {
  const { value } = props;
  const { isMobile } = usePlatformOS();
  return (
    <>
      {Array.isArray(value) ? (
        <ModuleArrayContent {...props} value={value} isMobile={isMobile} />
      ) : (
        <ModuleSingleContent {...props} value={value} />
      )}
      {props.dropdownArrow && (
        <ChevronDownIcon className={cn("h-2.5 w-2.5 flex-shrink-0", props.dropdownArrowClassName)} aria-hidden="true" />
      )}
    </>
  );
}

type ModuleArrayProps = Omit<ModuleButtonContentProps, "value"> & { value: string[]; isMobile: boolean };

function ModuleArrayContent(props: ModuleArrayProps) {
  if (props.showCount) return <ModuleCountContent {...props} />;
  if (props.value.length === 0)
    return (
      <>
        {!props.hideIcon && <ModuleIcon className="h-3 w-3 flex-shrink-0" />}
        <span className="flex-grow truncate text-left">{props.placeholder}</span>
      </>
    );

  return (
    <div className="flex max-w-full flex-grow flex-wrap items-center gap-2 truncate py-0.5">
      {props.value.map((moduleId) => (
        <ModuleValueChip key={moduleId} {...props} moduleId={moduleId} />
      ))}
    </div>
  );
}

function ModuleCountContent(props: ModuleArrayProps) {
  const value = props.value;
  const label =
    value.length === 0
      ? props.placeholder
      : value.length === 1
        ? props.modules?.find((module) => module.id === value[0])?.name || "module"
        : `${value.length} Modules`;
  return (
    <div className="relative flex max-w-full items-center gap-1">
      {!props.hideIcon && <ModuleIcon className="h-3 w-3 flex-shrink-0" />}
      {(value.length > 0 || !!props.placeholder) && <div className="max-w-40 truncate">{label}</div>}
    </div>
  );
}

function ModuleValueChip(props: ModuleArrayProps & { moduleId: string }) {
  const moduleDetails = props.modules?.find((module) => module.id === props.moduleId);
  return (
    <div
      className={cn("flex max-w-full items-center gap-1 rounded-sm bg-layer-1 py-1 text-secondary", props.className)}
    >
      {!props.hideIcon && <ModuleIcon className="h-2.5 w-2.5 flex-shrink-0" />}
      {!props.hideText && (
        <Tooltip
          tooltipHeading="Title"
          tooltipContent={moduleDetails?.name}
          disabled={!props.showTooltip}
          isMobile={props.isMobile}
          renderByDefault={false}
        >
          <span className="max-w-40 truncate text-11 font-medium">{moduleDetails?.name}</span>
        </Tooltip>
      )}
    </div>
  );
}

function ModuleSingleContent(props: Omit<ModuleButtonContentProps, "value"> & { value: string | null }) {
  const name = props.value ? props.modules?.find((module) => module.id === props.value)?.name : props.placeholder;
  return (
    <>
      {!props.hideIcon && <ModuleIcon className="h-3 w-3 flex-shrink-0" />}
      {!props.hideText && <span className="flex-grow truncate text-left">{name}</span>}
    </>
  );
}
