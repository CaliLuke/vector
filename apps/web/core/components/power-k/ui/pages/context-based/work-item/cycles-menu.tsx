/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// plane types
import type { ICycle, TIssue } from "@plane/types";
import { Spinner } from "@plane/ui";
// components
import { PowerKCyclesMenu } from "@/components/power-k/menus/cycles";
// hooks
import { useProjectCycles } from "@/hooks/use-cycles";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";

type Props = {
  handleSelect: (cycle: ICycle) => void;
  workItemDetails: TIssue;
};

export const PowerKWorkItemCyclesMenu = function PowerKWorkItemCyclesMenu(props: Props) {
  const { handleSelect, workItemDetails } = props;
  const { workspaceSlug } = useParams({ strict: false });
  const catalogue = useProjectCycles(workspaceSlug, workItemDetails.project_id ?? undefined);
  if (catalogue.isError || catalogue.isAccessDenied)
    return <CatalogueReadFailure label="cycles" retry={catalogue.retry} />;
  if (!catalogue.cycles) return <Spinner />;
  const filteredCyclesList = catalogue.cycles;

  return <PowerKCyclesMenu cycles={filteredCyclesList} onSelect={handleSelect} value={workItemDetails.cycle_id} />;
};
