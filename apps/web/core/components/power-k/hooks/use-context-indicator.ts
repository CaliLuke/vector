/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { getPageName } from "@plane/utils/page";
// hooks
import { useCycleDetails } from "@/hooks/use-cycles";
import { useModuleDetails } from "@/hooks/use-modules";
// local imports
import type { TPowerKContextType } from "../core/types";
import { useParams } from "@tanstack/react-router";
import { useProjectPage } from "@/hooks/use-project-pages";

type TArgs = {
  activeContext: TPowerKContextType | null;
};

export const useContextIndicator = (args: TArgs): string | null => {
  const { activeContext } = args;
  // navigation
  const {
    workspaceSlug,
    projectId,
    workItem: workItemIdentifier,
    cycleId,
    moduleId,
    pageId,
  } = useParams({ strict: false });
  // store hooks
  const { cycle } = useCycleDetails(workspaceSlug, projectId, cycleId);
  const { module } = useModuleDetails(workspaceSlug, projectId, moduleId);
  const pageQuery = useProjectPage(workspaceSlug, projectId, activeContext === "page" ? pageId : undefined, false);
  let indicator: string | undefined | null = null;

  switch (activeContext) {
    case "work-item": {
      indicator = workItemIdentifier ? workItemIdentifier.toString() : null;
      break;
    }
    case "cycle": {
      const cycleDetails = cycleId ? cycle : null;
      indicator = cycleDetails?.name;
      break;
    }
    case "module": {
      const moduleDetails = moduleId ? module : null;
      indicator = moduleDetails?.name;
      break;
    }
    case "page": {
      indicator = getPageName(pageQuery.data?.name);
      break;
    }
    default: {
      indicator = null;
    }
  }

  return indicator ?? null;
};
