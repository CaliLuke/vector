/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery } from "@tanstack/react-query";
import { instanceInfoOptions } from "@/queries/session";
// plane imports
import { useSearchParams } from "react-router";
import { useTheme } from "next-themes";
import { API_BASE_URL } from "@plane/constants";
import type { TOAuthConfigs, TOAuthOption } from "@plane/types";
// assets
import giteaLogo from "@/app/assets/logos/gitea-logo.svg?url";
import githubLightLogo from "@/app/assets/logos/github-black.png?url";
import githubDarkLogo from "@/app/assets/logos/github-dark.svg?url";
import gitlabLogo from "@/app/assets/logos/gitlab-logo.svg?url";
import googleLogo from "@/app/assets/logos/google-logo.svg?url";
// hooks

export const useCoreOAuthConfig = (oauthActionText: string): TOAuthConfigs => {
  //router
  const [searchParams] = useSearchParams();
  // query params
  const next_path = searchParams.get("next_path");
  // theme
  const { resolvedTheme } = useTheme();
  const { data: instanceInfo } = useQuery(instanceInfoOptions);
  const config = instanceInfo?.config;
  // derived values
  const providers = [
    { id: "google", name: "Google", logo: googleLogo, enabled: config?.is_google_enabled },
    {
      id: "github",
      name: "GitHub",
      logo: resolvedTheme === "dark" ? githubLightLogo : githubDarkLogo,
      enabled: config?.is_github_enabled,
    },
    { id: "gitlab", name: "GitLab", logo: gitlabLogo, enabled: config?.is_gitlab_enabled },
    { id: "gitea", name: "Gitea", logo: giteaLogo, enabled: config?.is_gitea_enabled },
  ];
  const oAuthOptions: TOAuthOption[] = providers.map(({ id, name, logo, enabled }) => ({
    id,
    text: `${oauthActionText} with ${name}`,
    enabled,
    icon: <img src={logo} height={18} width={18} alt={name} />,
    onClick: () => {
      const query = next_path ? `?${new URLSearchParams({ next_path })}` : "";
      window.location.assign(`${API_BASE_URL}/auth/${id}/${query}`);
    },
  }));

  return {
    isOAuthEnabled: oAuthOptions.some((option) => option.enabled),
    oAuthOptions,
  };
};
