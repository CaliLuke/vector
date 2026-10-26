/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTheme } from "next-themes";
import { Loader, ToggleSwitch } from "@plane/ui";
import { setPromiseToast } from "@plane/propel/toast";
import { Button } from "@plane/propel/button";
import { instanceConfigurationsOptions, updateConfigurationsOptions } from "@/queries/instance";
import { PageWrapper } from "@/components/common/page-wrapper";
import { AuthenticationMethodCard } from "./authentication-method-card";
import { OAuthConfigurationForm } from "./oauth-configuration-form";
import { oauthSettings } from "./oauth-settings";
import type { OAuthProvider } from "./oauth-settings";

export function OAuthSetupPage({ provider }: { provider: OAuthProvider }) {
  const settings = oauthSettings[provider];
  const query = useQuery(instanceConfigurationsOptions);
  const client = useQueryClient();
  const update = useMutation(updateConfigurationsOptions(client));
  const { resolvedTheme } = useTheme();
  const enabled = query.data?.[settings.enabledKey] === "1";
  const toggle = () => {
    const request = update.mutateAsync({ [settings.enabledKey]: enabled ? "0" : "1" });
    setPromiseToast(request, {
      loading: "Saving configuration",
      success: {
        title: "Configuration saved",
        message: () => `${settings.name} authentication is now ${enabled ? "disabled" : "active"}.`,
      },
      error: { title: "Error", message: () => "Failed to save configuration" },
    });
    void request.catch(console.error);
  };
  return (
    <PageWrapper
      customHeader={
        <AuthenticationMethodCard
          name={settings.name}
          description={`Allow members to log in or sign up for Vector with their ${settings.name} accounts.`}
          icon={
            <img
              src={resolvedTheme === "dark" ? (settings.darkLogo ?? settings.logo) : settings.logo}
              height={24}
              width={24}
              alt={`${settings.name} logo`}
            />
          }
          config={
            <ToggleSwitch value={enabled} onChange={toggle} size="sm" disabled={update.isPending || !query.data} />
          }
          disabled={update.isPending || !query.data}
          withBorder={false}
        />
      }
    >
      {query.data ? (
        <OAuthConfigurationForm key={provider} settings={settings} config={query.data} save={update.mutateAsync} />
      ) : query.isError ? (
        <div role="alert">
          Unable to load authentication settings.{" "}
          <Button
            onClick={() => {
              void query.refetch();
            }}
          >
            Retry
          </Button>
        </div>
      ) : (
        <Loader className="space-y-8">
          {Array.from({ length: 5 }, (_, index) => (
            <Loader.Item key={index} height="50px" />
          ))}
        </Loader>
      )}
    </PageWrapper>
  );
}
