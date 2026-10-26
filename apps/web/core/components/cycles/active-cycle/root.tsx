/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTheme } from "next-themes";
import { Disclosure } from "@headlessui/react";
import { EmptyStateDetailed } from "@plane/propel/empty-state";
// plane imports
import { useTranslation } from "@plane/i18n";
import type { TWorkItemFilterCondition } from "@plane/shared-state";
import type { ICycle } from "@plane/types";
import { Row } from "@plane/ui";
// assets
import darkActiveCycleAsset from "@/app/assets/empty-state/cycle/active-dark.webp?url";
import lightActiveCycleAsset from "@/app/assets/empty-state/cycle/active-light.webp?url";
// components
import { ActiveCycleStats } from "@/components/cycles/active-cycle/cycle-stats";
import { ActiveCycleProductivity } from "@/components/cycles/active-cycle/productivity";
import { ActiveCycleProgress } from "@/components/cycles/active-cycle/progress";
import useCyclesDetails from "@/components/cycles/active-cycle/use-cycles-details";
import { CycleListGroupHeader } from "@/components/cycles/list/cycle-list-group-header";
import { CyclesListItem } from "@/components/cycles/list/cycles-list-item";
// hooks
import { useProjectCycles } from "@/hooks/use-cycles";
import { activeCycle as selectActiveCycle } from "@/queries/cycle-module-selectors";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";

interface IActiveCycleDetails {
  workspaceSlug: string;
  projectId: string;
  cycleId?: string;
  showHeader?: boolean;
}

type ActiveCyclesComponentProps = {
  cycleId: string | null | undefined;
  activeCycle: ICycle | null;
  activeCycleResolvedPath: string;
  workspaceSlug: string;
  projectId: string;
  handleFiltersUpdate: (filters: TWorkItemFilterCondition[]) => void;
};

const ActiveCyclesComponent = function ActiveCyclesComponent({
  cycleId,
  activeCycle,
  activeCycleResolvedPath: _activeCycleResolvedPath,
  workspaceSlug,
  projectId,
  handleFiltersUpdate,
}: ActiveCyclesComponentProps) {
  const { t } = useTranslation();

  if (!cycleId || !activeCycle) {
    return (
      <EmptyStateDetailed
        assetKey="cycle"
        title={t("project_cycles.empty_state.active.title")}
        description={t("project_cycles.empty_state.active.description")}
        rootClassName="py-10 h-auto"
      />
    );
  }

  return (
    <div className="flex flex-col border-b border-subtle">
      <CyclesListItem
        key={cycleId}
        cycle={activeCycle}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        className="!border-b-transparent"
      />
      <Row className="bg-surface-1 pt-3 pb-6">
        <div className="grid grid-cols-1 gap-3 bg-surface-1 lg:grid-cols-2 xl:grid-cols-3">
          <ActiveCycleProgress
            handleFiltersUpdate={handleFiltersUpdate}
            projectId={projectId}
            workspaceSlug={workspaceSlug}
            cycle={activeCycle}
          />
          <ActiveCycleProductivity workspaceSlug={workspaceSlug} projectId={projectId} cycle={activeCycle} />
          <ActiveCycleStats
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            cycle={activeCycle}
            cycleId={cycleId}
            handleFiltersUpdate={handleFiltersUpdate}
          />
        </div>
      </Row>
    </div>
  );
};

export const ActiveCycleRoot = function ActiveCycleRoot(props: IActiveCycleDetails) {
  const { workspaceSlug, projectId, cycleId: propsCycleId, showHeader = true } = props;
  // theme hook
  const { resolvedTheme } = useTheme();
  // plane hooks
  const { t } = useTranslation();
  // store hooks
  const catalogue = useProjectCycles(workspaceSlug, projectId);
  // derived values
  const cycleId = propsCycleId ?? selectActiveCycle(catalogue.cycles)?.id;
  const activeCycleResolvedPath = resolvedTheme === "light" ? lightActiveCycleAsset : darkActiveCycleAsset;
  // fetch cycle details
  const {
    handleFiltersUpdate: saveFilters,
    cycle: activeCycle,
    isError,
    isPending,
    retry,
  } = useCyclesDetails({ workspaceSlug, projectId, cycleId });

  const handleFiltersUpdate = (conditions: TWorkItemFilterCondition[]) => {
    void saveFilters(conditions).catch((error) => console.warn("Cycle filters could not be saved.", error));
  };
  if (catalogue.isError || catalogue.isAccessDenied || isError)
    return (
      <CatalogueReadFailure
        label="active cycle"
        retry={async () => {
          await catalogue.retry();
          await retry();
        }}
      />
    );
  if (catalogue.isPending || isPending) return <output>Loading active cycle…</output>;

  return (
    <>
      {showHeader ? (
        <Disclosure as="div" className="flex flex-shrink-0 flex-col" defaultOpen>
          {({ open }) => (
            <>
              <Disclosure.Button className="sticky top-0 z-[2] w-full flex-shrink-0 cursor-pointer border-b border-subtle bg-layer-1">
                <CycleListGroupHeader title={t("project_cycles.active_cycle.label")} type="current" isExpanded={open} />
              </Disclosure.Button>
              <Disclosure.Panel>
                <ActiveCyclesComponent
                  cycleId={cycleId}
                  activeCycle={activeCycle}
                  activeCycleResolvedPath={activeCycleResolvedPath}
                  workspaceSlug={workspaceSlug}
                  projectId={projectId}
                  handleFiltersUpdate={handleFiltersUpdate}
                />
              </Disclosure.Panel>
            </>
          )}
        </Disclosure>
      ) : (
        <ActiveCyclesComponent
          cycleId={cycleId}
          activeCycle={activeCycle}
          activeCycleResolvedPath={activeCycleResolvedPath}
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          handleFiltersUpdate={handleFiltersUpdate}
        />
      )}
    </>
  );
};
