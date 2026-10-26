/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane types
import type { ICycle } from "@plane/types";
import { Spinner } from "@plane/ui";
// components
import type { TPowerKContext } from "@/components/power-k/core/types";
import { PowerKCyclesMenu } from "@/components/power-k/menus/cycles";
// hooks
import { useProjectCycles } from "@/hooks/use-cycles";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";

type Props = {
  context: TPowerKContext;
  handleSelect: (cycle: ICycle) => void;
};

export const PowerKOpenProjectCyclesMenu = function PowerKOpenProjectCyclesMenu(props: Props) {
  const { context, handleSelect } = props;
  const catalogue = useProjectCycles(
    typeof context.params.workspaceSlug === "string" ? context.params.workspaceSlug : undefined,
    typeof context.params.projectId === "string" ? context.params.projectId : undefined
  );
  if (catalogue.isError || catalogue.isAccessDenied)
    return <CatalogueReadFailure label="cycles" retry={catalogue.retry} />;
  if (!catalogue.cycles) return <Spinner />;
  const cyclesList = catalogue.cycles;

  return <PowerKCyclesMenu cycles={cyclesList} onSelect={handleSelect} />;
};
