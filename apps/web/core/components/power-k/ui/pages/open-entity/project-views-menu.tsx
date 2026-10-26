/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane types
import type { IProjectView } from "@plane/types";
import { Spinner } from "@plane/ui";
// components
import type { TPowerKContext } from "@/components/power-k/core/types";
// hooks
import { PowerKViewsMenu } from "@/components/power-k/menus/views";
import { useProjectViews } from "@/hooks/use-project-views";

type Props = {
  context: TPowerKContext;
  handleSelect: (view: IProjectView) => void;
};

export function PowerKOpenProjectViewsMenu(props: Props) {
  const { context, handleSelect } = props;
  // store hooks
  // derived values
  const projectId = context.params.projectId?.toString();
  const workspaceSlug = context.params.workspaceSlug?.toString();
  const viewsQuery = useProjectViews(workspaceSlug, projectId);
  const viewsList = viewsQuery.views ?? [];

  if (viewsQuery.isPending) return <Spinner />;
  if (viewsQuery.isError || viewsQuery.isDenied)
    return (
      <button type="button" onClick={() => void viewsQuery.retry()}>
        Project views could not be loaded. Retry
      </button>
    );

  return <PowerKViewsMenu views={viewsList} onSelect={handleSelect} />;
}
