/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { EUserPermissions } from "@plane/types";

export const WORKSPACE_PARTIAL_PROJECTS = (workspaceSlug: string | undefined) =>
  ["WORKSPACE_PARTIAL_PROJECTS", workspaceSlug] as const;

export const WORKSPACE_MODULES = (workspaceSlug: string | undefined) => ["WORKSPACE_MODULES", workspaceSlug] as const;

export const WORKSPACE_CYCLES = (workspaceSlug: string | undefined) => ["WORKSPACE_CYCLES", workspaceSlug] as const;

export const WORKSPACE_LABELS = (workspaceSlug: string | undefined) => ["WORKSPACE_LABELS", workspaceSlug] as const;

export const WORKSPACE_ESTIMATES = (workspaceSlug: string | undefined) =>
  ["WORKSPACE_ESTIMATES", workspaceSlug] as const;

export const WORKSPACE_INVITATION = (workspaceSlug: string | undefined, invitationId: string | undefined) =>
  ["WORKSPACE_INVITATION", workspaceSlug, invitationId] as const;

export const WORKSPACE_FAVORITE = (workspaceSlug: string | undefined) => ["WORKSPACE_FAVORITE", workspaceSlug] as const;

export const WORKSPACE_STATES = (workspaceSlug: string | undefined) => ["WORKSPACE_STATES", workspaceSlug] as const;

export const PROJECT_GITHUB_REPOSITORY = (
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  integrationId: string
) => ["PROJECT_GITHUB_REPOSITORY", workspaceSlug, projectId, integrationId] as const;

export const CYCLE_ISSUES_WITH_PARAMS = (
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  cycleId: string | undefined,
  params: { priority: string; per_page: number }
) => ["CYCLE_ISSUES_WITH_PARAMS", workspaceSlug, projectId, cycleId, params] as const;

export const APP_INTEGRATIONS = ["APP_INTEGRATIONS"] as const;

export const WORKSPACE_INTEGRATIONS = (workspaceSlug: string | undefined) =>
  ["WORKSPACE_INTEGRATIONS", workspaceSlug] as const;

export const EXPORT_SERVICES_LIST = (
  workspaceSlug: string | undefined,
  cursor: string | undefined,
  per_page: string | undefined
) => ["EXPORT_SERVICES_LIST", workspaceSlug, cursor, per_page] as const;

export const SLACK_CHANNEL_INFO = (
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  integrationId: string
) => ["SLACK_CHANNEL_INFO", workspaceSlug, projectId, integrationId] as const;

export const USER_PROFILE_DATA = (workspaceSlug: string | undefined, userId: string | undefined) =>
  ["USER_PROFILE_DATA", workspaceSlug, userId] as const;

export const USER_PROFILE_ACTIVITY = (
  workspaceSlug: string | undefined,
  userId: string | undefined,
  params: {
    cursor?: string;
    per_page: number;
  }
) => ["USER_PROFILE_ACTIVITY", workspaceSlug, userId, params] as const;

export const USER_PROFILE_PROJECT_SEGREGATION = (workspaceSlug: string | undefined, userId: string | undefined) =>
  ["USER_PROFILE_PROJECT_SEGREGATION", workspaceSlug, userId] as const;

export const API_TOKENS_LIST = ["API_TOKENS_LIST"] as const;

export const PROJECT_DETAILS = (workspaceSlug: string | undefined, projectId: string | undefined) =>
  ["PROJECT_DETAILS", workspaceSlug, projectId] as const;

export const PROJECT_LABELS = (
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  projectRole: EUserPermissions | undefined
) => ["PROJECT_LABELS", workspaceSlug, projectId, projectRole] as const;

export const PROJECT_STATES = (
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  projectRole: EUserPermissions | undefined
) => ["PROJECT_STATES", workspaceSlug, projectId, projectRole] as const;

export const PROJECT_INTAKE_STATE = (
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  projectRole: EUserPermissions | undefined
) => ["PROJECT_INTAKE_STATE", workspaceSlug, projectId, projectRole] as const;

export const PROJECT_ESTIMATES = (
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  projectRole: EUserPermissions | undefined
) => ["PROJECT_ESTIMATES", workspaceSlug, projectId, projectRole] as const;

export const PROJECT_ALL_CYCLES = (
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  projectRole: EUserPermissions | undefined
) => ["PROJECT_ALL_CYCLES", workspaceSlug, projectId, projectRole] as const;

export const PROJECT_MODULES = (
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  projectRole: EUserPermissions | undefined
) => ["PROJECT_MODULES", workspaceSlug, projectId, projectRole] as const;

export const PROJECT_VIEWS = (
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  projectRole: EUserPermissions | undefined
) => ["PROJECT_VIEWS", workspaceSlug, projectId, projectRole] as const;
