/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { useCallback, useMemo } from "react";

// plane imports
import { EUserPermissions } from "@plane/constants";
import { EUserWorkspaceRoles } from "@plane/types";
import type { IProjectUserPropertiesResponse, TProject } from "@plane/types";
import { Button } from "@plane/propel/button";
// components
import { ProjectAccessRestriction } from "@/components/auth-screens/project/project-access-restriction";
// hooks
import { useProjectEstimates } from "@/hooks/use-estimates";
import { useProjectCycles } from "@/hooks/use-cycles";
import { useProjectLabels } from "@/hooks/use-labels";
import { useProjectModules } from "@/hooks/use-modules";
import { useProjectDetail } from "@/hooks/use-projects";
import { useProjectStates } from "@/hooks/use-project-states";
import { useJoinProject, useProjectPermissions } from "@/hooks/use-permissions";
import { usePatchProjectUserProperties, useProjectUserProperties } from "@/hooks/use-project-user-properties";
import {
  projectIssueFiltersFromProperties,
  projectIssueRequestInputFromProperties,
} from "@/helpers/issue-list/request";
import type { ProjectUserPropertiesPatch } from "@/queries/project-user-properties";
import { ProjectIssuePreferencesContext } from "@/hooks/use-project-issue-preferences-context";

interface IProjectAuthWrapper {
  workspaceSlug: string;
  projectId: string;
  children: ReactNode;
  isLoading?: boolean;
}

function projectIdentifierFor(project: TProject | undefined, projectId: string) {
  return project?.id === projectId ? project.identifier : undefined;
}

export const ProjectAuthWrapper = function ProjectAuthWrapper(props: IProjectAuthWrapper) {
  const { workspaceSlug, projectId, children, isLoading: isParentLoading = false } = props;
  // store hooks
  const permissions = useProjectPermissions(workspaceSlug, projectId);
  const joinProject = useJoinProject(workspaceSlug, projectId);
  // derived values
  const hasPermissionToCurrentProject = permissions.can([
    EUserPermissions.ADMIN,
    EUserPermissions.MEMBER,
    EUserPermissions.GUEST,
  ]);
  useProjectLabels(workspaceSlug, projectId);
  const isWorkspaceAdmin = permissions.workspaceRole === EUserWorkspaceRoles.ADMIN;
  // fetching project details
  const projectDetails = useProjectDetail(workspaceSlug, projectId, true, true);
  const canLoadProject = hasPermissionToCurrentProject && projectDetails.project !== undefined;
  const projectPropertiesQuery = useProjectEntities(
    workspaceSlug,
    projectId,
    hasPermissionToCurrentProject,
    canLoadProject,
    projectDetails.project
  );
  const canWriteProjectPreferences =
    hasPermissionToCurrentProject && projectPropertiesQuery.data !== undefined && !projectPropertiesQuery.isError;
  const { mutateAsync: patchProjectProperties } = usePatchProjectUserProperties(
    workspaceSlug,
    projectId,
    canWriteProjectPreferences
  );
  const saveProjectPreferences = useCallback(
    async (patch: ProjectUserPropertiesPatch): Promise<IProjectUserPropertiesResponse> => {
      if (!canWriteProjectPreferences) throw new Error("Project preferences are not ready to update.");
      return patchProjectProperties(patch);
    },
    [canWriteProjectPreferences, patchProjectProperties]
  );
  const projectIdentifier = projectIdentifierFor(projectDetails.project, projectId);
  const preferenceContext = useMemo(
    () => ({
      workspaceSlug,
      projectId,
      properties: projectPropertiesQuery.data,
      issueFilters: projectPropertiesQuery.data
        ? projectIssueFiltersFromProperties(projectPropertiesQuery.data)
        : undefined,
      requestInput: projectPropertiesQuery.data
        ? projectIssueRequestInputFromProperties(
            workspaceSlug,
            projectId,
            projectPropertiesQuery.data,
            projectIdentifier
          )
        : undefined,
      isPending: projectPropertiesQuery.isPending,
      isError: projectPropertiesQuery.isError,
      refetch: projectPropertiesQuery.refetch,
      save: saveProjectPreferences,
    }),
    [
      projectId,
      projectPropertiesQuery.data,
      projectPropertiesQuery.isError,
      projectPropertiesQuery.isPending,
      projectPropertiesQuery.refetch,
      projectIdentifier,
      saveProjectPreferences,
      workspaceSlug,
    ]
  );

  const retryAccess = () => {
    void projectDetails.retryAccess();
  };
  const handleJoinProject = () => {
    joinProject.mutate(undefined, {
      onSuccess: () => {
        void projectDetails.retryAccess();
      },
    });
  };
  const errorStatus = projectErrorStatus(projectDetails.error);
  const accessState = projectAccessState({
    isParentLoading,
    permissions,
    projectDetails,
    errorStatus,
    hasPermissionToCurrentProject,
    hasJoined: joinProject.isSuccess,
  });
  if (accessState === "loading") return null;

  if (accessState === "unavailable" || accessState === "joined") {
    return (
      <div role="alert" className="flex h-full flex-col items-center justify-center gap-3">
        <p>
          {joinProject.isSuccess
            ? "Project joined. Unable to load project access yet."
            : "Unable to load project access. Please try again."}
        </p>
        <Button
          disabled={
            permissions.membershipQuery.isFetching || permissions.rolesQuery.isFetching || projectDetails.isFetching
          }
          onClick={retryAccess}
        >
          Retry
        </Button>
      </div>
    );
  }

  if (accessState === "restricted") {
    return (
      <>
        {joinProject.isError && (
          <p role="alert" className="text-center text-danger-primary">
            Unable to join this project. Please try again.
          </p>
        )}
        <ProjectAccessRestriction
          errorStatusCode={errorStatus}
          isWorkspaceAdmin={isWorkspaceAdmin}
          handleJoinProject={handleJoinProject}
          isJoinButtonDisabled={joinProject.isPending}
        />
      </>
    );
  }

  return (
    <>
      {projectPropertiesQuery.isError && (
        <div role="alert" className="flex items-center gap-3">
          <p>Unable to load project preferences.</p>
          <Button disabled={projectPropertiesQuery.isFetching} onClick={() => void projectPropertiesQuery.refetch()}>
            Retry
          </Button>
        </div>
      )}
      <ProjectIssuePreferencesContext.Provider value={preferenceContext}>
        {children}
      </ProjectIssuePreferencesContext.Provider>
    </>
  );
};

function useProjectEntities(
  workspaceSlug: string,
  projectId: string,
  canAccessProject: boolean,
  canLoadProject: boolean,
  project: TProject | undefined
) {
  useProjectStates(workspaceSlug, projectId, Boolean(project?.inbox_view), canLoadProject);
  useProjectEstimates(canLoadProject ? workspaceSlug : undefined, canLoadProject ? projectId : undefined);
  // Fetch the full project-preferences response into its native account/scope Query cache.
  const projectPropertiesQuery = useProjectUserProperties(workspaceSlug, projectId, {
    canAccess: canAccessProject,
    fetchEnabled: canLoadProject,
  });
  useProjectCycles(canLoadProject ? workspaceSlug : undefined, canLoadProject ? projectId : undefined);
  useProjectModules(canLoadProject ? workspaceSlug : undefined, canLoadProject ? projectId : undefined);
  return projectPropertiesQuery;
}

function projectErrorStatus(error: unknown) {
  if (error && typeof error === "object" && "status" in error && typeof error.status === "number") return error.status;
  return undefined;
}

function projectAccessState({
  isParentLoading,
  permissions,
  projectDetails,
  errorStatus,
  hasPermissionToCurrentProject,
  hasJoined,
}: {
  isParentLoading: boolean;
  permissions: Pick<ReturnType<typeof useProjectPermissions>, "isPending" | "isError">;
  projectDetails: { isPending: boolean; isError: boolean; isSuccess: boolean };
  errorStatus: number | undefined;
  hasPermissionToCurrentProject: boolean;
  hasJoined: boolean;
}) {
  if (isParentLoading || permissions.isPending || projectDetails.isPending) return "loading";
  if (permissions.isError || (projectDetails.isError && ![403, 404, 409].includes(errorStatus ?? 0)))
    return "unavailable";
  if (hasJoined && (!hasPermissionToCurrentProject || !projectDetails.isSuccess)) return "joined";
  if (!hasPermissionToCurrentProject || projectDetails.isError) return "restricted";
  return "ready";
}
