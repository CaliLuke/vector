/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

import { useQuery, skipToken, useQueryClient, useMutation } from "@tanstack/react-query";
// types
import type { IWorkspaceIntegration } from "@plane/types";
// ui
import { Loader } from "@plane/ui";
import { setToast, TOAST_TYPE } from "@plane/propel/toast";
// query keys
import { SLACK_CHANNEL_INFO } from "@plane/constants";
// hooks
import { instanceConfigOptions } from "@/queries/instance";
import useIntegrationPopup from "@/hooks/use-integration-popup";
// services
import { AppInstallationService } from "@/services/app_installation.service";

type Props = {
  integration: IWorkspaceIntegration;
};

const appInstallationService = new AppInstallationService();

export function SelectChannel({ integration }: Props) {
  const queryClient = useQueryClient();
  // store hooks
  const { data: config } = useQuery(instanceConfigOptions);
  const { workspaceSlug, projectId } = useParams({ strict: false });

  const { startAuth } = useIntegrationPopup({
    provider: "slackChannel",
    stateParams: integration.id,
    slack_client_id: config?.slack_client_id || "",
  });

  const { data: projectIntegration } = useQuery({
    queryKey: SLACK_CHANNEL_INFO(workspaceSlug, projectId, integration.id),
    queryFn:
      workspaceSlug && projectId && integration.id
        ? () => appInstallationService.getSlackChannelDetail(workspaceSlug, projectId, integration.id)
        : skipToken,
  });

  const slackChannel = projectIntegration?.find((channel) => channel.project === projectId);
  const { mutate: removeChannel, isPending: isRemoving } = useMutation({
    mutationFn: async () => {
      if (!workspaceSlug || !projectId || !slackChannel) return;
      await appInstallationService.removeSlackChannel(workspaceSlug, projectId, integration.id, slackChannel.id);
      await queryClient.invalidateQueries({ queryKey: SLACK_CHANNEL_INFO(workspaceSlug, projectId, integration.id) });
    },
    onError: () =>
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Failed to disconnect channel",
        message: "Please try again.",
      }),
  });

  return (
    <>
      {projectIntegration ? (
        <button
          type="button"
          className={`bg-gray-700 relative inline-flex h-4 w-6 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none`}
          role="switch"
          aria-checked={!!slackChannel}
          disabled={isRemoving}
          onClick={() => {
            if (slackChannel) removeChannel();
            else startAuth();
          }}
        >
          <span
            aria-hidden="true"
            className={`shadow inline-block h-2 w-2 transform self-center rounded-full bg-white ring-0 transition duration-200 ease-in-out ${
              slackChannel ? "translate-x-3" : "translate-x-0"
            }`}
          />
        </button>
      ) : (
        <Loader>
          <Loader.Item height="35px" width="150px" />
        </Loader>
      )}
    </>
  );
}
