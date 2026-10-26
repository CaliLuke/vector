/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback } from "react";

import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { orderJoinedProjects } from "@plane/utils/project";
import type { TProject } from "@plane/types";
import { useProjectSortOrderMutation, useWorkspaceProjectDetails } from "@/hooks/use-projects";

type TWorkspaceProjectReorder = {
  workspaceSlug: string;
  joinedProjectIds: string[];
};

export const useWorkspaceProjectReordering = ({ workspaceSlug, joinedProjectIds }: TWorkspaceProjectReorder) => {
  const { t } = useTranslation();
  const { projects } = useWorkspaceProjectDetails(workspaceSlug);
  const updateSortOrder = useProjectSortOrderMutation(workspaceSlug);

  return useCallback(
    (sourceId: string | undefined, destinationId: string | undefined, shouldDropAtEnd: boolean) => {
      if (!sourceId || !destinationId || sourceId === destinationId) return;

      const joinedProjects: TProject[] = joinedProjectIds.flatMap((projectId) => {
        const project = projects?.find((candidate) => candidate.id === projectId);
        return project ? [project] : [];
      });
      const orderedProjectIds = joinedProjects.map((project) => project.id);

      const sourceIndex = orderedProjectIds.indexOf(sourceId);
      const destinationIndex = shouldDropAtEnd ? orderedProjectIds.length : orderedProjectIds.indexOf(destinationId);
      if (sourceIndex < 0 || destinationIndex < 0) return;

      const updatedSortOrder = orderJoinedProjects(sourceIndex, destinationIndex, sourceId, joinedProjects);
      if (updatedSortOrder === undefined) return;

      void updateSortOrder.mutateAsync({ projectId: sourceId, sort_order: updatedSortOrder }).catch(() => {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("error"),
          message: t("something_went_wrong"),
        });
      });
    },
    [joinedProjectIds, projects, t, updateSortOrder]
  );
};
