/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import React from "react";

import { useInfiniteQuery } from "@tanstack/react-query";
import type { IGithubRepository, IWorkspaceIntegration } from "@plane/types";
// services
// ui
import { CustomSearchSelect } from "@plane/ui";
// helpers
import { truncateText } from "@plane/utils/string";
import { API_BASE_URL } from "@plane/constants";
import { ProjectService } from "@/services/project";
// types

type Props = {
  integration: IWorkspaceIntegration;
  value: string | null;
  label: string | React.ReactNode;
  onChange: (repo: IGithubRepository | undefined) => void;
  characterLimit?: number;
};

const projectService = new ProjectService();

export function SelectRepository(props: Props) {
  const { integration, value, label, onChange, characterLimit = 25 } = props;
  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });

  const { data, fetchNextPage, hasNextPage, isFetching } = useInfiniteQuery({
    queryKey: ["GITHUB_REPOSITORIES", workspaceSlug, integration.id],
    initialPageParam: 1,
    queryFn: ({ pageParam }) =>
      projectService.getGithubRepositories(
        `${API_BASE_URL}/api/workspaces/${workspaceSlug}/workspace-integrations/${integration.id}/github-repositories/?page=${pageParam}`
      ),
    getNextPageParam: (lastPage, pages, lastPageParam) => {
      const loadedCount = pages.reduce((count, page) => count + page.repositories.length, 0);
      return lastPage.repositories.length > 0 && loadedCount < lastPage.total_count ? lastPageParam + 1 : undefined;
    },
  });
  const userRepositories = data?.pages.flatMap((page) => page.repositories).filter((repo) => repo?.id) ?? [];

  const options =
    userRepositories.map((repo) => ({
      value: repo.id,
      query: repo.full_name,
      content: <p>{truncateText(repo.full_name, characterLimit)}</p>,
    })) ?? [];

  if (userRepositories.length < 1) return null;

  return (
    <CustomSearchSelect
      value={value}
      options={options}
      onChange={(val: string) => {
        const repo = userRepositories.find((repo) => repo.id === val);

        onChange(repo);
      }}
      label={label}
      footerOption={
        <>
          {hasNextPage && (
            <button
              type="button"
              className="w-full p-1 text-center text-10 text-secondary hover:bg-layer-1"
              onClick={() => void fetchNextPage()}
              disabled={isFetching}
            >
              {isFetching ? "Loading..." : "Click to load more..."}
            </button>
          )}
        </>
      }
      optionsClassName="w-48"
    />
  );
}
