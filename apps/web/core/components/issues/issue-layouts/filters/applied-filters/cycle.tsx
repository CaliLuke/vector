/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

import { CloseIcon, CycleGroupIcon } from "@plane/propel/icons";
import { cycleGroup } from "@/queries/cycle-module-selectors";
// hooks
import { useProjectCycles } from "@/hooks/use-cycles";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";
// ui
// types

type Props = {
  handleRemove: (val: string) => void;
  values: string[];
  editable: boolean | undefined;
};

export const AppliedCycleFilters = function AppliedCycleFilters(props: Props) {
  const { handleRemove, values, editable } = props;
  // store hooks
  const { workspaceSlug, projectId } = useParams({ strict: false });
  const catalogue = useProjectCycles(workspaceSlug, projectId);

  return (
    <>
      {(catalogue.isError || catalogue.isAccessDenied) && (
        <CatalogueReadFailure label="cycles" retry={catalogue.retry} />
      )}
      {values.map((cycleId) => {
        const cycleDetails = catalogue.cycles?.find((row) => row.id === cycleId) ?? null;

        if (!cycleDetails) return null;

        const cycleStatus = cycleGroup(cycleDetails.status);

        return (
          <div key={cycleId} className="flex items-center gap-1 truncate rounded-sm bg-layer-1 p-1 text-11">
            <CycleGroupIcon cycleGroup={cycleStatus} className="h-3 w-3 flex-shrink-0" />
            <span className="truncate normal-case">{cycleDetails.name}</span>
            {editable && (
              <button
                type="button"
                className="grid place-items-center text-tertiary hover:text-secondary"
                onClick={() => handleRemove(cycleId)}
              >
                <CloseIcon height={10} width={10} strokeWidth={2} />
              </button>
            )}
          </div>
        );
      })}
    </>
  );
};
