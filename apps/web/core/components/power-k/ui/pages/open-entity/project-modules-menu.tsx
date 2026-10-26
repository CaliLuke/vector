/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane types
import type { IModule } from "@plane/types";
import { Spinner } from "@plane/ui";
// components
import type { TPowerKContext } from "@/components/power-k/core/types";
import { PowerKModulesMenu } from "@/components/power-k/menus/modules";
// hooks
import { useProjectModules } from "@/hooks/use-modules";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";

type Props = {
  context: TPowerKContext;
  handleSelect: (module: IModule) => void;
};

export const PowerKOpenProjectModulesMenu = function PowerKOpenProjectModulesMenu(props: Props) {
  const { context, handleSelect } = props;
  const catalogue = useProjectModules(
    typeof context.params.workspaceSlug === "string" ? context.params.workspaceSlug : undefined,
    typeof context.params.projectId === "string" ? context.params.projectId : undefined
  );
  if (catalogue.isError || catalogue.isAccessDenied)
    return <CatalogueReadFailure label="modules" retry={catalogue.retry} />;
  if (!catalogue.modules) return <Spinner />;
  const modulesList = catalogue.modules;

  return <PowerKModulesMenu modules={modulesList} onSelect={handleSelect} />;
};
