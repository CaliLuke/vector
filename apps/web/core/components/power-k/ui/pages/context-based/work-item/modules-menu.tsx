/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// plane types
import type { IModule, TIssue } from "@plane/types";
import { Spinner } from "@plane/ui";
// components
import { PowerKModulesMenu } from "@/components/power-k/menus/modules";
// hooks
import { useProjectModules } from "@/hooks/use-modules";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";

type Props = {
  handleSelect: (module: IModule) => void;
  workItemDetails: TIssue;
};

export const PowerKWorkItemModulesMenu = function PowerKWorkItemModulesMenu(props: Props) {
  const { handleSelect, workItemDetails } = props;
  const { workspaceSlug } = useParams({ strict: false });
  const catalogue = useProjectModules(workspaceSlug, workItemDetails.project_id ?? undefined);
  if (catalogue.isError || catalogue.isAccessDenied)
    return <CatalogueReadFailure label="modules" retry={catalogue.retry} />;
  if (!catalogue.modules) return <Spinner />;
  const filteredModulesList = catalogue.modules;

  return (
    <PowerKModulesMenu modules={filteredModulesList} onSelect={handleSelect} value={workItemDetails.module_ids ?? []} />
  );
};
