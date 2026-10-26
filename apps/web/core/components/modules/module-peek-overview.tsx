/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLocation, useRouter } from "@tanstack/react-router";
import React, { useCallback, useEffect, useMemo } from "react";

// hooks
import { useModuleDetails } from "@/hooks/use-modules";
import { useCommittedScope } from "@/hooks/use-committed-scope";

// components
import { ModuleAnalyticsSidebar } from "./";
import { getModulePeekHref } from "./module-peek-navigation";

type Props = {
  projectId: string;
  workspaceSlug: string;
  isArchived?: boolean;
};

export const ModulePeekOverview = function ModulePeekOverview({ projectId, workspaceSlug, isArchived = false }: Props) {
  // router
  const router = useRouter();
  const pathname = useLocation({ select: (location) => location.pathname });
  const searchStr = useLocation({ select: (location) => location.searchStr });
  const searchParams = useMemo(() => new URLSearchParams(searchStr), [searchStr]);
  const peekModule = searchParams.get("peekModule");
  const detail = useModuleDetails(workspaceSlug, projectId, peekModule ?? undefined, isArchived);
  const current = useCommittedScope(JSON.stringify([detail.scope, peekModule]));
  // refs
  const ref = React.useRef(null);
  // store hooks

  const handleClose = useCallback(() => {
    void router.navigate({ href: getModulePeekHref(pathname, searchStr) });
  }, [pathname, router, searchStr]);

  useEffect(() => {
    if (detail.module === null && current()) handleClose();
  }, [detail.module, current, handleClose]);

  return (
    <>
      {peekModule && (
        <div
          ref={ref}
          className="absolute right-0 z-[9] flex h-full w-full max-w-[24rem] flex-shrink-0 flex-col gap-3.5 overflow-y-auto border-l border-subtle bg-surface-1 px-6 duration-300 md:relative"
          style={{
            boxShadow:
              "0px 1px 4px 0px rgba(0, 0, 0, 0.06), 0px 2px 4px 0px rgba(16, 24, 40, 0.06), 0px 1px 8px -1px rgba(16, 24, 40, 0.06)",
          }}
        >
          <ModuleAnalyticsSidebar
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            moduleId={peekModule?.toString() ?? ""}
            handleClose={handleClose}
            isArchived={isArchived}
          />
        </div>
      )}
    </>
  );
};
