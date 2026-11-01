/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";

// plane imports
import { ModuleStatusIcon } from "@plane/propel/icons";
import type { IModule } from "@plane/types";
// local imports
import { PowerKMenuBuilder } from "./builder";

type Props = {
  modules: IModule[];
  onSelect: (module: IModule) => void;
  value?: string[];
};

function getModuleIcon(module: Props["modules"][number]) {
  return <ModuleStatusIcon status={module.status ?? "backlog"} className="size-3.5 shrink-0" />;
}

export const PowerKModulesMenu = function PowerKModulesMenu({ modules, onSelect, value }: Props) {
  return (
    <PowerKMenuBuilder
      items={modules}
      getKey={(module) => module.id}
      getIconNode={getModuleIcon}
      getValue={(module) => module.name}
      getLabel={(module) => module.name}
      isSelected={(module) => !!value?.includes(module.id)}
      onSelect={onSelect}
      emptyText="No modules found"
    />
  );
};
