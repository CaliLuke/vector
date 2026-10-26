/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type {
  IFormattedInstanceConfiguration,
  TInstanceConfigurationKeys,
  TInstanceAuthenticationMethodKeys,
} from "@plane/types";
import type { SettingsInput } from "@/components/common/settings-form";
import type { TCopyField } from "@/components/common/copy-field";
import { CodeBlock } from "@/components/common/code-block";
import googleLogo from "@/app/assets/logos/google-logo.svg?url";
import githubLight from "@/app/assets/logos/github-black.png?url";
import githubDark from "@/app/assets/logos/github-white.png?url";
import gitlabLogo from "@/app/assets/logos/gitlab-logo.svg?url";
import giteaLogo from "@/app/assets/logos/gitea-logo.svg?url";

export type OAuthValues = Partial<IFormattedInstanceConfiguration>;
export type OAuthProvider = "google" | "github" | "gitlab" | "gitea";
export type OAuthSettings = {
  name: string;
  enabledKey: TInstanceAuthenticationMethodKeys;
  syncKey: TInstanceConfigurationKeys;
  defaults: OAuthValues;
  logo: string;
  darkLogo?: string;
  fields: (SettingsInput & { key: TInstanceConfigurationKeys })[];
  serviceFields: (originURL: string, values: OAuthValues) => TCopyField[];
};

export const oauthSettings: Record<OAuthProvider, OAuthSettings> = {
  google: {
    name: "Google",
    enabledKey: "IS_GOOGLE_ENABLED",
    syncKey: "ENABLE_GOOGLE_SYNC",
    defaults: {},
    logo: googleLogo,
    fields: [
      {
        key: "GOOGLE_CLIENT_ID",
        type: "text",
        label: "Client ID",
        description: (
          <>
            Your client ID lives in your Google API Console.{" "}
            <a
              href="https://developers.google.com/identity/protocols/oauth2/javascript-implicit-flow#creatingcred"
              target="_blank"
              className="text-accent-primary hover:underline"
              rel="noreferrer"
              aria-label="Google OAuth client ID documentation"
            >
              Learn more
            </a>
          </>
        ),
        placeholder: "840195096245-0p2tstej9j5nc4l8o1ah2dqondscqc1g.apps.googleusercontent.com",
        required: true,
      },
      {
        key: "GOOGLE_CLIENT_SECRET",
        type: "password",
        label: "Client secret",
        description: (
          <>
            Your client secret should also be in your Google API Console.{" "}
            <a
              href="https://developers.google.com/identity/oauth2/web/guides/get-google-api-clientid"
              target="_blank"
              className="text-accent-primary hover:underline"
              rel="noreferrer"
              aria-label="Google OAuth client secret documentation"
            >
              Learn more
            </a>
          </>
        ),
        placeholder: "GOCShX-ADp4cI0kPqav1gGCBg5bE02E",
        required: true,
      },
    ],
    serviceFields: (originURL, _values) => [
      {
        key: "Origin_URL",
        label: "Origin URL",
        url: originURL,
        description: (
          <p>
            We will auto-generate this. Paste this into your{" "}
            <CodeBlock darkerShade>Authorized JavaScript origins</CodeBlock> field. For this OAuth client{" "}
            <a
              href="https://console.cloud.google.com/apis/credentials/oauthclient"
              target="_blank"
              className="text-accent-primary hover:underline"
              rel="noreferrer"
              aria-label="Google Cloud Console OAuth client credentials"
            >
              here.
            </a>
          </p>
        ),
      },
      {
        key: "Callback_URI",
        label: "Callback URI",
        url: `${originURL}/auth/google/callback/`,
        description: (
          <p>
            We will auto-generate this. Paste this into your <CodeBlock darkerShade>Authorized Redirect URI</CodeBlock>{" "}
            field. For this OAuth client{" "}
            <a
              href="https://console.cloud.google.com/apis/credentials/oauthclient"
              target="_blank"
              className="text-accent-primary hover:underline"
              rel="noreferrer"
              aria-label="Google Cloud Console OAuth client credentials"
            >
              here.
            </a>
          </p>
        ),
      },
    ],
  },
  github: {
    name: "GitHub",
    enabledKey: "IS_GITHUB_ENABLED",
    syncKey: "ENABLE_GITHUB_SYNC",
    defaults: {},
    logo: githubLight,
    darkLogo: githubDark,
    fields: [
      {
        key: "GITHUB_CLIENT_ID",
        type: "text",
        label: "Client ID",
        description: (
          <>
            You will get this from your{" "}
            <a
              href="https://github.com/settings/applications/new"
              target="_blank"
              className="text-accent-primary hover:underline"
              rel="noreferrer"
            >
              GitHub OAuth application settings.
            </a>
          </>
        ),
        placeholder: "70a44354520df8bd9bcd",
        required: true,
      },
      {
        key: "GITHUB_CLIENT_SECRET",
        type: "password",
        label: "Client secret",
        description: (
          <>
            Your client secret is also found in your{" "}
            <a
              href="https://github.com/settings/applications/new"
              target="_blank"
              className="text-accent-primary hover:underline"
              rel="noreferrer"
            >
              GitHub OAuth application settings.
            </a>
          </>
        ),
        placeholder: "9b0050f94ec1b744e32ce79ea4ffacd40d4119cb",
        required: true,
      },
      {
        key: "GITHUB_ORGANIZATION_ID",
        type: "text",
        label: "Organization ID",
        description: <>The organization github ID.</>,
        placeholder: "123456789",
        required: false,
      },
    ],
    serviceFields: (originURL, _values) => [
      {
        key: "Origin_URL",
        label: "Origin URL",
        url: originURL,
        description: (
          <>
            We will auto-generate this. Paste this into the <CodeBlock darkerShade>Authorized origin URL</CodeBlock>{" "}
            field{" "}
            <a
              href="https://github.com/settings/applications/new"
              target="_blank"
              className="text-accent-primary hover:underline"
              rel="noreferrer"
              aria-label="GitHub OAuth application settings"
            >
              here.
            </a>
          </>
        ),
      },
      {
        key: "Callback_URI",
        label: "Callback URI",
        url: `${originURL}/auth/github/callback/`,
        description: (
          <>
            We will auto-generate this. Paste this into your <CodeBlock darkerShade>Authorized Callback URI</CodeBlock>{" "}
            field{" "}
            <a
              href="https://github.com/settings/applications/new"
              target="_blank"
              className="text-accent-primary hover:underline"
              rel="noreferrer"
              aria-label="GitHub OAuth application settings"
            >
              here.
            </a>
          </>
        ),
      },
    ],
  },
  gitlab: {
    name: "GitLab",
    enabledKey: "IS_GITLAB_ENABLED",
    syncKey: "ENABLE_GITLAB_SYNC",
    defaults: { GITLAB_HOST: "https://gitlab.com" },
    logo: gitlabLogo,
    fields: [
      {
        key: "GITLAB_HOST",
        type: "text",
        label: "Host",
        description: (
          <>
            This is either https://gitlab.com or the <CodeBlock>domain.tld</CodeBlock> where you host GitLab.
          </>
        ),
        placeholder: "https://gitlab.com",
        required: true,
      },
      {
        key: "GITLAB_CLIENT_ID",
        type: "text",
        label: "Application ID",
        description: (
          <>
            Get this from your{" "}
            <a
              href="https://docs.gitlab.com/ee/integration/oauth_provider.html"
              target="_blank"
              className="text-accent-primary hover:underline"
              rel="noreferrer"
            >
              GitLab OAuth application settings
            </a>
            .
          </>
        ),
        placeholder: "c2ef2e7fc4e9d15aa7630f5637d59e8e4a27ff01dceebdb26b0d267b9adcf3c3",
        required: true,
      },
      {
        key: "GITLAB_CLIENT_SECRET",
        type: "password",
        label: "Secret",
        description: (
          <>
            The client secret is also found in your{" "}
            <a
              href="https://docs.gitlab.com/ee/integration/oauth_provider.html"
              target="_blank"
              className="text-accent-primary hover:underline"
              rel="noreferrer"
            >
              GitLab OAuth application settings
            </a>
            .
          </>
        ),
        placeholder: "Enter your GitLab application secret",
        required: true,
      },
    ],
    serviceFields: (originURL, _values) => [
      {
        key: "Callback_URL",
        label: "Callback URL",
        url: `${originURL}/auth/gitlab/callback/`,
        description: (
          <>
            We will auto-generate this. Paste this into the <CodeBlock darkerShade>Redirect URI</CodeBlock> field of
            your{" "}
            <a
              href="https://docs.gitlab.com/ee/integration/oauth_provider.html"
              target="_blank"
              className="text-accent-primary hover:underline"
              rel="noreferrer"
            >
              GitLab OAuth application
            </a>
            .
          </>
        ),
      },
    ],
  },
  gitea: {
    name: "Gitea",
    enabledKey: "IS_GITEA_ENABLED",
    syncKey: "ENABLE_GITEA_SYNC",
    defaults: { GITEA_HOST: "https://gitea.com" },
    logo: giteaLogo,
    fields: [
      {
        key: "GITEA_HOST",
        type: "text",
        label: "Gitea Host",
        description: (
          <>Use the URL of your Gitea instance. For the official Gitea instance, use &quot;https://gitea.com&quot;.</>
        ),
        placeholder: "https://gitea.com",
        required: true,
      },
      {
        key: "GITEA_CLIENT_ID",
        type: "text",
        label: "Client ID",
        description: (
          <>
            You will get this from your{" "}
            <a
              href="https://gitea.com/user/settings/applications"
              target="_blank"
              className="text-accent-primary hover:underline"
              rel="noreferrer"
            >
              Gitea OAuth application settings.
            </a>
          </>
        ),
        placeholder: "70a44354520df8bd9bcd",
        required: true,
      },
      {
        key: "GITEA_CLIENT_SECRET",
        type: "password",
        label: "Client secret",
        description: (
          <>
            Your client secret is also found in your{" "}
            <a
              href="https://gitea.com/user/settings/applications"
              target="_blank"
              className="text-accent-primary hover:underline"
              rel="noreferrer"
            >
              Gitea OAuth application settings.
            </a>
          </>
        ),
        placeholder: "9b0050f94ec1b744e32ce79ea4ffacd40d4119cb",
        required: true,
      },
    ],
    serviceFields: (originURL, values) => [
      {
        key: "Callback_URI",
        label: "Callback URI",
        url: `${originURL}/auth/gitea/callback/`,
        description: (
          <>
            We will auto-generate this. Paste this into your <CodeBlock darkerShade>Authorized Callback URI</CodeBlock>{" "}
            field{" "}
            <a
              href={`${values.GITEA_HOST || "https://gitea.com"}/user/settings/applications`}
              target="_blank"
              className="text-accent-primary hover:underline"
              rel="noreferrer"
              aria-label="Gitea OAuth application settings"
            >
              here.
            </a>
          </>
        ),
      },
    ],
  },
};
