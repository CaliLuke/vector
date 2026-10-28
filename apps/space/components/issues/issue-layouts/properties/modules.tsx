/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
import { useParams } from "react-router";

// plane ui
import { ModuleIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
// plane utils
import { cn } from "@plane/utils/common";
// hooks
import { modulesOptions } from "@/queries/lookups";

type Props = {
  moduleIds: string[] | undefined;
  shouldShowBorder?: boolean;
};

export const IssueBlockModules = function IssueBlockModules({ moduleIds, shouldShowBorder = true }: Props) {
  const { data: modules = [] } = useRecordCollection(modulesOptions(useParams().anchor));

  const selectedModules = modules.filter((item) => (moduleIds ?? []).includes(item.id));

  const modulesString = selectedModules.map((module) => module.name).join(", ");

  return (
    <div className="relative flex h-full flex-wrap items-center gap-1">
      <Tooltip tooltipHeading="Modules" tooltipContent={modulesString}>
        {selectedModules.length <= 1 ? (
          <div
            key={selectedModules?.[0]?.id}
            className={cn("flex h-full flex-shrink-0 cursor-default items-center rounded-md px-2.5 py-1 text-11", {
              "border-[0.5px] border-strong": shouldShowBorder,
            })}
          >
            <div className="flex items-center gap-1.5 text-secondary">
              <ModuleIcon className="h-3 w-3 flex-shrink-0" />
              <div className="text-11">{selectedModules?.[0]?.name ?? "No Modules"}</div>
            </div>
          </div>
        ) : (
          <div className="flex h-full flex-shrink-0 cursor-default items-center rounded-md border border-strong px-2.5 py-1 text-11">
            <div className="flex items-center gap-1.5 text-secondary">
              <div className="text-11">{selectedModules.length} Modules</div>
            </div>
          </div>
        )}
      </Tooltip>
    </div>
  );
};
