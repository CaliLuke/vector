/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  createWorkspaceOptions,
  deleteWorkspaceOptions,
  updateWorkspaceOptions,
  workspacesOptions,
} from "@/queries/workspaces";
import { useCurrentUserId } from "./use-current-user";

export function useWorkspaces() {
  return useQuery(workspacesOptions(useCurrentUserId()));
}

export function useWorkspaceBySlug(slug: string | undefined) {
  return useQuery({
    ...workspacesOptions(useCurrentUserId()),
    select: (workspaces) => workspaces.find((workspace) => workspace.slug === slug),
  });
}

export function useWorkspaceById(id: string | undefined) {
  return useQuery({
    ...workspacesOptions(useCurrentUserId()),
    select: (workspaces) => workspaces.find((workspace) => workspace.id === id),
  });
}

export function useCurrentWorkspace() {
  const { workspaceSlug } = useParams({ strict: false });
  return useWorkspaceBySlug(typeof workspaceSlug === "string" ? workspaceSlug : undefined);
}

export function useCreateWorkspace() {
  return useMutation(createWorkspaceOptions(useQueryClient(), useCurrentUserId()));
}

export function useUpdateWorkspace(slug: string | undefined) {
  return useMutation(updateWorkspaceOptions(useQueryClient(), useCurrentUserId(), slug));
}

export function useDeleteWorkspace(slug: string | undefined) {
  return useMutation(deleteWorkspaceOptions(useQueryClient(), useCurrentUserId(), slug));
}
