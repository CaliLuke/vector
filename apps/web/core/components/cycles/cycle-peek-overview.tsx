/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLocation, useRouter } from "@tanstack/react-router";
import React, { useCallback, useEffect, useMemo } from "react";

// hooks
import { generateQueryParams } from "@plane/utils/router";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { useCycleDetails } from "@/hooks/use-cycles";
import { useWorkspaceCycleUniverse } from "@/hooks/use-cycles";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";

// components
import { CycleDetailsSidebar } from "./analytics-sidebar";

type Props = {
  projectId?: string;
  workspaceSlug: string;
  isArchived?: boolean;
};

export const CyclePeekOverview = function CyclePeekOverview(props: Props) {
  const { projectId: propsProjectId, workspaceSlug, isArchived } = props;
  // router
  const router = useRouter();
  const pathname = useLocation({ select: (location) => location.pathname });
  const searchStr = useLocation({ select: (location) => location.searchStr });
  const searchParams = useMemo(() => new URLSearchParams(searchStr), [searchStr]);
  const peekCycle = searchParams.get("peekCycle");
  // refs
  const ref = React.useRef(null);
  // store hooks
  const catalogue = useWorkspaceCycleUniverse(propsProjectId ? undefined : workspaceSlug);
  // derived values
  const cycleDetails = catalogue.cycles?.find((cycle) => cycle.id === peekCycle);
  const projectId = propsProjectId || cycleDetails?.project_id;
  const detail = useCycleDetails(workspaceSlug, projectId, peekCycle ?? undefined, isArchived);
  const current = useCommittedScope(JSON.stringify([detail.scope, peekCycle]));

  const handleClose = useCallback(() => {
    const query = generateQueryParams(searchParams, ["peekCycle"]);
    void router.navigate({ href: `${pathname}?${query}` });
  }, [pathname, router, searchParams]);

  useEffect(() => {
    if (current() && (detail.cycle === null || (!propsProjectId && catalogue.cycles !== undefined && !cycleDetails)))
      handleClose();
  }, [detail.cycle, current, propsProjectId, catalogue.cycles, cycleDetails, handleClose]);
  return (
    <>
      {peekCycle && !propsProjectId && (catalogue.isError || catalogue.isAccessDenied) && (
        <CatalogueReadFailure label="cycles" retry={catalogue.retry} />
      )}
      {peekCycle && projectId && (
        <div
          ref={ref}
          className="fixed right-0 z-[9] flex h-full w-full max-w-[21.5rem] flex-shrink-0 flex-col gap-3.5 overflow-y-auto border-l border-subtle bg-surface-1 px-4 duration-300 md:relative"
          style={{
            boxShadow:
              "0px 1px 4px 0px rgba(0, 0, 0, 0.06), 0px 2px 4px 0px rgba(16, 24, 40, 0.06), 0px 1px 8px -1px rgba(16, 24, 40, 0.06)",
          }}
        >
          <CycleDetailsSidebar
            handleClose={handleClose}
            isArchived={isArchived}
            projectId={projectId}
            workspaceSlug={workspaceSlug}
            cycleId={peekCycle}
          />
        </div>
      )}
    </>
  );
};
