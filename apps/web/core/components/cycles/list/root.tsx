/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import type { ICycle } from "@plane/types";

import { Disclosure } from "@headlessui/react";
// components
import { useTranslation } from "@plane/i18n";
import { ContentWrapper, ERowVariant } from "@plane/ui";
import { ListLayout } from "@/components/core/list";
// local imports
import { CyclePeekOverview } from "../cycle-peek-overview";
import { CycleListGroupHeader } from "./cycle-list-group-header";
import { CyclesListMap } from "./cycles-list-map";
import { ActiveCycleRoot } from "../active-cycle/root";

export interface ICyclesList {
  completedCycles: ICycle[];
  upcomingCycles?: ICycle[] | undefined;
  cycles: ICycle[];
  workspaceSlug: string;
  projectId: string;
  isArchived?: boolean;
}

export const CyclesList = function CyclesList(props: ICyclesList) {
  const { completedCycles, upcomingCycles, cycles, workspaceSlug, projectId, isArchived = false } = props;
  const { t } = useTranslation();

  return (
    <ContentWrapper variant={ERowVariant.HUGGING} className="flex-row">
      <ListLayout>
        {isArchived ? (
          <>
            <CyclesListMap cycles={cycles} projectId={projectId} workspaceSlug={workspaceSlug} />
          </>
        ) : (
          <>
            <ActiveCycleRoot workspaceSlug={workspaceSlug} projectId={projectId} />

            {upcomingCycles && (
              <Disclosure as="div" className="flex flex-shrink-0 flex-col" defaultOpen>
                {({ open }) => (
                  <>
                    <Disclosure.Button className="sticky top-0 z-[2] w-full flex-shrink-0 cursor-pointer border-b border-subtle bg-layer-1">
                      <CycleListGroupHeader
                        title={t("project_cycles.upcoming_cycle.label")}
                        type="upcoming"
                        count={upcomingCycles.length}
                        showCount
                        isExpanded={open}
                      />
                    </Disclosure.Button>
                    <Disclosure.Panel>
                      <CyclesListMap cycles={upcomingCycles} projectId={projectId} workspaceSlug={workspaceSlug} />
                    </Disclosure.Panel>
                  </>
                )}
              </Disclosure>
            )}
            <Disclosure as="div" className="flex flex-shrink-0 flex-col pb-7">
              {({ open }) => (
                <>
                  <Disclosure.Button className="sticky top-0 z-2 w-full flex-shrink-0 cursor-pointer border-b border-subtle bg-layer-1">
                    <CycleListGroupHeader
                      title={t("project_cycles.completed_cycle.label")}
                      type="completed"
                      count={completedCycles.length}
                      showCount
                      isExpanded={open}
                    />
                  </Disclosure.Button>
                  <Disclosure.Panel>
                    <CyclesListMap cycles={completedCycles} projectId={projectId} workspaceSlug={workspaceSlug} />
                  </Disclosure.Panel>
                </>
              )}
            </Disclosure>
          </>
        )}
      </ListLayout>
      <CyclePeekOverview projectId={projectId} workspaceSlug={workspaceSlug} isArchived={isArchived} />
    </ContentWrapper>
  );
};
