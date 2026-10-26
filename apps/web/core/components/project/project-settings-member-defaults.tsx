/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
// plane imports
import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IProject, IUserLite } from "@plane/types";
import { Loader, ToggleSwitch } from "@plane/ui";
// constants
// hooks
import { useProjectDetail, useProjectMutations } from "@/hooks/use-projects";
import { useProjectPermissions } from "@/hooks/use-permissions";
// local imports
import { MemberSelect } from "./member-select";

const memberId = (member: string | IUserLite | null | undefined) => (typeof member === "string" ? member : member?.id);

type TDefaultSettingItemProps = {
  title: string;
  description: string;
  children: ReactNode;
};

function DefaultSettingItem({ title, description, children }: TDefaultSettingItemProps) {
  return (
    <div className="flex items-center justify-between gap-x-2">
      <div className="flex flex-col gap-0.5">
        <h4 className="text-13 font-medium">{title}</h4>
        <p className="text-11 text-tertiary">{description}</p>
      </div>
      <div className="w-full max-w-48 sm:max-w-64">{children}</div>
    </div>
  );
}

type TProjectMemberSettingProps = {
  title: string;
  description: string;
  value: string | undefined;
  workspaceSlug: string;
  projectId: string;
  isDisabled: boolean;
  onChange: (value: string) => void;
  isLoading: boolean;
};

function ProjectMemberSetting({
  title,
  description,
  value,
  workspaceSlug,
  projectId,
  isDisabled,
  onChange,
  isLoading,
}: TProjectMemberSettingProps) {
  return (
    <DefaultSettingItem title={title} description={description}>
      {isLoading ? (
        <Loader className="h-9 w-full">
          <Loader.Item width="100%" height="100%" />
        </Loader>
      ) : (
        <MemberSelect
          value={value}
          onChange={onChange}
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          isDisabled={isDisabled}
        />
      )}
    </DefaultSettingItem>
  );
}

type TProjectSettingsMemberDefaultsProps = {
  workspaceSlug: string;
  projectId: string;
};

export function ProjectSettingsMemberDefaults(props: TProjectSettingsMemberDefaultsProps) {
  const { workspaceSlug, projectId } = props;
  // plane hooks
  const { t } = useTranslation();
  // store hooks
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  const projectQuery = useProjectDetail(workspaceSlug, projectId);
  const projectMutation = useProjectMutations(workspaceSlug, projectId);
  // derived values
  const isAdmin = canProject([EUserPermissions.ADMIN]);
  const project = projectQuery.project;
  const isError = projectQuery.isError;
  if (projectQuery.accessDenied) return <div role="alert">You do not have access to this project.</div>;
  if (isError)
    return (
      <div role="alert">
        Unable to load project settings.{" "}
        <button
          type="button"
          onClick={() => {
            void projectQuery.retryAccess();
          }}
        >
          Retry
        </button>
      </div>
    );
  const updateSettings = async (data: Partial<IProject>) => {
    try {
      await projectMutation.mutateAsync({ type: "update", projectId, data });
      setToast({
        title: t("success"),
        type: TOAST_TYPE.SUCCESS,
        message: t("project_settings.general.toast.success"),
      });
    } catch {
      setToast({
        title: t("error"),
        type: TOAST_TYPE.ERROR,
        message: "Unable to update project settings. Please try again.",
      });
    }
  };
  return (
    <div className="my-6 flex flex-col gap-y-6">
      <ProjectMemberSetting
        title="Project Lead"
        description="Select the project lead for the project."
        value={memberId(project?.project_lead)}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        isDisabled={!isAdmin || projectMutation.isPending}
        onChange={(value) => void updateSettings({ project_lead: value === "none" ? null : value })}
        isLoading={!project}
      />
      <ProjectMemberSetting
        title="Default Assignee"
        description="Select the default assignee for the project."
        value={memberId(project?.default_assignee)}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        isDisabled={!isAdmin || projectMutation.isPending}
        onChange={(value) => void updateSettings({ default_assignee: value === "none" ? null : value })}
        isLoading={!project}
      />
      {project && (
        <DefaultSettingItem
          title="Guest access"
          description="This will allow guests to have view access to all the project work items."
        >
          <div className="flex items-center justify-end">
            <ToggleSwitch
              value={!!project?.guest_view_all_features}
              onChange={(value) => void updateSettings({ guest_view_all_features: value })}
              disabled={!isAdmin || projectMutation.isPending}
              size="sm"
            />
          </div>
        </DefaultSettingItem>
      )}
    </div>
  );
}
