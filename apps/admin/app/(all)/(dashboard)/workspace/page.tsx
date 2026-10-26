/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useInfiniteQuery } from "@tanstack/react-query";
import { instanceWorkspacesOptions } from "@/queries/workspaces";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { instanceConfigurationsOptions, updateConfigurationsOptions } from "@/queries/instance";
import { notifyConfigurationSave } from "@/helpers/configuration-save";
import { Link } from "react-router";
import { Loader as LoaderIcon } from "lucide-react";
// types
import { Button, getButtonStyling } from "@plane/propel/button";
import type { TInstanceConfigurationKeys } from "@plane/types";
import { Loader, ToggleSwitch } from "@plane/ui";
import { cn } from "@plane/utils/common";
// components
import { PageWrapper } from "@/components/common/page-wrapper";
import { WorkspaceListItem } from "@/components/workspace/list-item";
// hooks
// types
import type { Route } from "./+types/page";

const WorkspaceManagementPage = function WorkspaceManagementPage(_props: Route.ComponentProps) {
  // states
  const { data: formattedConfig } = useQuery(instanceConfigurationsOptions);
  const queryClient = useQueryClient();
  const { mutateAsync: updateInstanceConfigurations, isPending: isSubmitting } = useMutation(
    updateConfigurationsOptions(queryClient)
  );
  const workspacesQuery = useInfiniteQuery(instanceWorkspacesOptions);
  const workspaces = workspacesQuery.data?.pages.flatMap((page) => page.results) ?? [];
  const disableWorkspaceCreation = formattedConfig?.DISABLE_WORKSPACE_CREATION ?? "";

  const updateConfig = (key: TInstanceConfigurationKeys, value: string) => {
    notifyConfigurationSave(updateInstanceConfigurations({ [key]: value }));
  };

  return (
    <PageWrapper
      header={{
        title: "Workspaces on this instance",
        description: "See all workspaces and control who can create them.",
      }}
    >
      <div className="space-y-3">
        {workspacesQuery.isError && (
          <div role="alert">
            Unable to load workspaces.{" "}
            <Button
              onClick={() => {
                void workspacesQuery.refetch();
              }}
            >
              Retry
            </Button>
          </div>
        )}
        {formattedConfig ? (
          <div className={cn("flex w-full items-center gap-14 rounded-sm")}>
            <div className="flex grow items-center gap-4">
              <div className="grow">
                <div className="pb-1 text-16 font-medium">Prevent anyone else from creating a workspace.</div>
                <div className={cn("text-11 leading-5 font-regular text-tertiary")}>
                  Toggling this on will let only you create workspaces. You will have to invite users to new workspaces.
                </div>
              </div>
            </div>
            <div className={`shrink-0 pr-4 ${isSubmitting && "opacity-70"}`}>
              <div className="flex items-center gap-4">
                <ToggleSwitch
                  value={Boolean(parseInt(disableWorkspaceCreation))}
                  onChange={() => {
                    if (parseInt(disableWorkspaceCreation)) {
                      updateConfig("DISABLE_WORKSPACE_CREATION", "0");
                    } else {
                      updateConfig("DISABLE_WORKSPACE_CREATION", "1");
                    }
                  }}
                  size="sm"
                  disabled={isSubmitting}
                />
              </div>
            </div>
          </div>
        ) : (
          <Loader>
            <Loader.Item height="50px" width="100%" />
          </Loader>
        )}
        {!workspacesQuery.isPending ? (
          <>
            <div className="flex items-center justify-between gap-2 pt-6">
              <div className="flex flex-col items-start gap-x-2">
                <div className="flex items-center gap-2 text-16 font-medium">
                  All workspaces on this instance <span className="text-tertiary">• {workspaces.length}</span>
                  {workspacesQuery.isFetching && <LoaderIcon className="h-4 w-4 animate-spin" />}
                </div>
                <div className={cn("text-11 leading-5 font-regular text-tertiary")}>
                  You can&apos;t yet delete workspaces and you can only go to the workspace if you are an Admin or a
                  Member.
                </div>
              </div>
              <div className="flex items-center gap-2">
                <Link to="/workspace/create" className={getButtonStyling("primary", "base")}>
                  Create workspace
                </Link>
              </div>
            </div>
            <div className="flex flex-col gap-4 py-2">
              {workspaces.map((workspace) => (
                <WorkspaceListItem key={workspace.id} workspace={workspace} />
              ))}
            </div>
            {workspacesQuery.hasNextPage && (
              <div className="flex justify-center">
                <Button
                  variant="link"
                  size="lg"
                  onClick={() => {
                    void workspacesQuery.fetchNextPage();
                  }}
                  disabled={workspacesQuery.isFetchingNextPage}
                >
                  Load more
                  {workspacesQuery.isFetchingNextPage && <LoaderIcon className="h-3 w-3 animate-spin" />}
                </Button>
              </div>
            )}
          </>
        ) : (
          <Loader className="space-y-10 py-8">
            <Loader.Item height="24px" width="20%" />
            <Loader.Item height="92px" width="100%" />
            <Loader.Item height="92px" width="100%" />
            <Loader.Item height="92px" width="100%" />
          </Loader>
        )}
      </div>
    </PageWrapper>
  );
};

export const meta: Route.MetaFunction = () => [{ title: "Workspace Management - God Mode" }];

export default WorkspaceManagementPage;
