/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
import { useParams } from "@tanstack/react-router";
import { skipToken, useQueryClient } from "@tanstack/react-query";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IGithubRepository, IWorkspaceIntegration } from "@plane/types";
// assets
import GithubLogo from "@/app/assets/logos/github-square.png?url";
import SlackLogo from "@/app/assets/services/slack.png?url";
// components
import { SelectChannel } from "@/components/integration/slack/select-channel";
import { SelectRepository } from "@/components/integration/github/select-repository";
// constants
import { PROJECT_GITHUB_REPOSITORY } from "@plane/constants";
// services
import { ProjectService } from "@/services/project";

type Props = {
  integration: IWorkspaceIntegration;
};

const integrationDetails: Record<string, { logo: string; description: string }> = {
  github: {
    logo: GithubLogo,
    description: "Select GitHub repository to enable sync.",
  },
  slack: {
    logo: SlackLogo,
    description: "Get regular updates and control which notification you want to receive.",
  },
};

// services
const projectService = new ProjectService();
type SyncedGithubRepository = { repo_detail: { owner: string; name: string } };

export function IntegrationCard({ integration }: Props) {
  const queryClient = useQueryClient();
  const { workspaceSlug, projectId } = useParams({ strict: false });

  const { data: syncedGithubRepository } = useRecordCollection(
    {
      queryKey: PROJECT_GITHUB_REPOSITORY(workspaceSlug, projectId, integration.id),
      queryFn:
        workspaceSlug && projectId
          ? (): Promise<SyncedGithubRepository[]> =>
              projectService.getProjectGithubRepository(workspaceSlug, projectId, integration.id)
          : skipToken,
    },
    (row) => `${row.repo_detail.owner}/${row.repo_detail.name}`
  );

  const handleChange = async (repo: IGithubRepository | undefined) => {
    if (!workspaceSlug || !projectId || !repo) return;
    const separatorIndex = repo.full_name.indexOf("/");
    if (separatorIndex < 1 || separatorIndex === repo.full_name.length - 1) return;
    const owner = repo.full_name.slice(0, separatorIndex);
    const name = repo.full_name.slice(separatorIndex + 1);

    try {
      await projectService.syncGithubRepository(workspaceSlug, projectId, integration.id, {
        name,
        owner,
        repository_id: repo.id,
        url: repo.html_url,
      });
      const queryKey = PROJECT_GITHUB_REPOSITORY(workspaceSlug, projectId, integration.id);
      await queryClient.cancelQueries({ queryKey });
      await queryClient.invalidateQueries({ queryKey });
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Success!",
        message: `${owner}/${name} repository synced with the project successfully.`,
      });
    } catch (error) {
      console.error(error);
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Repository could not be synced with the project. Please try again.",
      });
    }
  };

  return (
    <>
      {integration && (
        <div className="flex items-center justify-between gap-2 border-b border-subtle bg-surface-1 px-4 py-6">
          <div className="flex items-start gap-4">
            <div className="h-10 w-10 flex-shrink-0">
              <img
                src={integrationDetails[integration.integration_detail.provider].logo}
                className="h-full w-full object-cover"
                alt={`${integration.integration_detail.title} Logo`}
              />
            </div>
            <div>
              <h3 className="flex items-center gap-4 text-13 font-medium">{integration.integration_detail.title}</h3>
              <p className="text-13 tracking-tight text-secondary">
                {integrationDetails[integration.integration_detail.provider].description}
              </p>
            </div>
          </div>
          {integration.integration_detail.provider === "github" && (
            <SelectRepository
              integration={integration}
              value={
                syncedGithubRepository && syncedGithubRepository.length > 0
                  ? `${syncedGithubRepository[0].repo_detail.owner}/${syncedGithubRepository[0].repo_detail.name}`
                  : null
              }
              label={
                syncedGithubRepository && syncedGithubRepository.length > 0
                  ? `${syncedGithubRepository[0].repo_detail.owner}/${syncedGithubRepository[0].repo_detail.name}`
                  : "Select Repository"
              }
              onChange={(repository) => void handleChange(repository)}
            />
          )}
          {integration.integration_detail.provider === "slack" && <SelectChannel integration={integration} />}
        </div>
      )}
    </>
  );
}
