/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
import sortBy from "lodash-es/sortBy.js";
import { useMutation, useMutationState, useQuery, useQueryClient } from "@tanstack/react-query";
import type { DefaultError, Mutation } from "@tanstack/react-query";
import type { IProjectUserPropertiesResponse } from "@plane/types";
import {
  applyProjectUserPropertiesPatches,
  patchProjectUserPropertiesOptions,
  projectUserPropertiesKeys,
  projectUserPropertiesOptions,
} from "../queries/project-user-properties";
import type {
  ProjectUserPropertiesPatch,
  ProjectUserPropertiesQueryAccess,
  ProjectUserPropertiesScope,
} from "@/queries/project-user-properties";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";

type ProjectPropertiesMutation = Mutation<IProjectUserPropertiesResponse, DefaultError, ProjectUserPropertiesPatch>;

type PendingPatch = {
  variables: ProjectUserPropertiesPatch;
  submittedAt: number;
  mutationId: number;
};

export function useProjectUserProperties(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  access: ProjectUserPropertiesQueryAccess
) {
  const client = useQueryClient();
  const accountId = useCurrentUserId();
  const generation = useIdentityGeneration(client);
  const scope: ProjectUserPropertiesScope = { accountId, workspaceSlug, projectId };
  const query = useQuery(projectUserPropertiesOptions(client, scope, access));
  const mutationKey = projectUserPropertiesKeys.patch(scope, generation);
  const pending = useMutationState<PendingPatch, ProjectPropertiesMutation>({
    filters: { mutationKey, exact: true, status: "pending" },
    select: (mutation) => ({
      variables: mutation.state.variables ?? {},
      submittedAt: mutation.state.submittedAt,
      mutationId: mutation.mutationId,
    }),
  });
  const data = useMemo(
    () =>
      access.canAccess && query.data
        ? applyProjectUserPropertiesPatches(
            query.data,
            sortBy(pending, [(mutation) => mutation.submittedAt, (mutation) => mutation.mutationId]).map(
              (mutation) => mutation.variables
            )
          )
        : undefined,
    [access.canAccess, pending, query.data]
  );

  return { ...query, data };
}

export function usePatchProjectUserProperties(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  canAccess: boolean
) {
  const client = useQueryClient();
  const accountId = useCurrentUserId();
  return useMutation(patchProjectUserPropertiesOptions(client, { accountId, workspaceSlug, projectId }, canAccess));
}
