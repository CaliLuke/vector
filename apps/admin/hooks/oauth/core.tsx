/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { KeyRound, Mails } from "lucide-react";
// types
import type { TCoreInstanceAuthenticationModeKeys, TInstanceAuthenticationModes } from "@plane/types";
// assets
import giteaLogo from "@/app/assets/logos/gitea-logo.svg?url";
import githubLightModeImage from "@/app/assets/logos/github-black.png?url";
import githubDarkModeImage from "@/app/assets/logos/github-white.png?url";
import gitlabLogo from "@/app/assets/logos/gitlab-logo.svg?url";
import googleLogo from "@/app/assets/logos/google-logo.svg?url";
// components
import { ConfigurationControl } from "@/components/authentication/configuration-control";
import type { TGetAuthenticationModeProps } from "./types";

// Authentication methods
export const getCoreAuthenticationModesMap: (
  props: TGetAuthenticationModeProps
) => Record<TCoreInstanceAuthenticationModeKeys, TInstanceAuthenticationModes> = ({
  disabled,
  updateConfig,
  resolvedTheme,
  config,
}) => ({
  "unique-codes": {
    key: "unique-codes",
    name: "Unique codes",
    description:
      "Log in or sign up for Vector using codes sent via email. You need to have set up SMTP to use this method.",
    icon: <Mails className="h-6 w-6 p-0.5 text-tertiary" />,
    config: (
      <ConfigurationControl
        config={config}
        configKey="ENABLE_MAGIC_LINK_LOGIN"
        disabled={disabled}
        updateConfig={updateConfig}
      />
    ),
    enabledConfigKey: "ENABLE_MAGIC_LINK_LOGIN",
  },
  "passwords-login": {
    key: "passwords-login",
    name: "Passwords",
    description: "Allow members to create accounts with passwords and use it with their email addresses to sign in.",
    icon: <KeyRound className="h-6 w-6 p-0.5 text-tertiary" />,
    config: (
      <ConfigurationControl
        config={config}
        configKey="ENABLE_EMAIL_PASSWORD"
        disabled={disabled}
        updateConfig={updateConfig}
      />
    ),
    enabledConfigKey: "ENABLE_EMAIL_PASSWORD",
  },
  google: {
    key: "google",
    name: "Google",
    description: "Allow members to log in or sign up for Vector with their Google accounts.",
    icon: <img src={googleLogo} height={20} width={20} alt="Google Logo" />,
    config: (
      <ConfigurationControl
        config={config}
        configKey="IS_GOOGLE_ENABLED"
        provider="google"
        disabled={disabled}
        updateConfig={updateConfig}
      />
    ),
    enabledConfigKey: "IS_GOOGLE_ENABLED",
  },
  github: {
    key: "github",
    name: "GitHub",
    description: "Allow members to log in or sign up for Vector with their GitHub accounts.",
    icon: (
      <img
        src={resolvedTheme === "dark" ? githubDarkModeImage : githubLightModeImage}
        height={20}
        width={20}
        alt="GitHub Logo"
      />
    ),
    config: (
      <ConfigurationControl
        config={config}
        configKey="IS_GITHUB_ENABLED"
        provider="github"
        disabled={disabled}
        updateConfig={updateConfig}
      />
    ),
    enabledConfigKey: "IS_GITHUB_ENABLED",
  },
  gitlab: {
    key: "gitlab",
    name: "GitLab",
    description: "Allow members to log in or sign up for Vector with their GitLab accounts.",
    icon: <img src={gitlabLogo} height={20} width={20} alt="GitLab Logo" />,
    config: (
      <ConfigurationControl
        config={config}
        configKey="IS_GITLAB_ENABLED"
        provider="gitlab"
        disabled={disabled}
        updateConfig={updateConfig}
      />
    ),
    enabledConfigKey: "IS_GITLAB_ENABLED",
  },
  gitea: {
    key: "gitea",
    name: "Gitea",
    description: "Allow members to log in or sign up for Vector with their Gitea accounts.",
    icon: <img src={giteaLogo} height={20} width={20} alt="Gitea Logo" />,
    config: (
      <ConfigurationControl
        config={config}
        configKey="IS_GITEA_ENABLED"
        provider="gitea"
        disabled={disabled}
        updateConfig={updateConfig}
      />
    ),
    enabledConfigKey: "IS_GITEA_ENABLED",
  },
});
