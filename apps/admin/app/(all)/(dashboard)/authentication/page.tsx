/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { instanceConfigurationsOptions, updateConfigurationsOptions } from "@/queries/instance";
import { notifyConfigurationSave } from "@/helpers/configuration-save";
import { useTheme } from "next-themes";
// plane internal packages
import { setToast, TOAST_TYPE } from "@plane/propel/toast";
import type { TInstanceConfigurationKeys } from "@plane/types";
import { Loader, ToggleSwitch } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { resolveGeneralTheme } from "@plane/utils/theme-legacy";
// components
import { PageWrapper } from "@/components/common/page-wrapper";
import { AuthenticationMethodCard } from "@/components/authentication/authentication-method-card";
// helpers
import { canDisableAuthMethod } from "@/helpers/authentication";
// hooks
import { useAuthenticationModes } from "@/hooks/oauth";
// types
import type { Route } from "./+types/page";

const InstanceAuthenticationPage = function InstanceAuthenticationPage(_props: Route.ComponentProps) {
  // theme
  const { resolvedTheme: resolvedThemeAdmin } = useTheme();
  const resolvedTheme = resolveGeneralTheme(resolvedThemeAdmin);
  // state
  const { data: formattedConfig } = useQuery(instanceConfigurationsOptions);
  const queryClient = useQueryClient();
  const { mutateAsync: updateInstanceConfigurations, isPending: isSubmitting } = useMutation(
    updateConfigurationsOptions(queryClient)
  );
  // derived values
  const enableSignUpConfig = formattedConfig?.ENABLE_SIGNUP ?? "";

  function updateConfig(key: TInstanceConfigurationKeys, value: string): void {
    // Check if trying to disable (value === "0")
    if (value === "0") {
      // Check if this key is an authentication method key
      const isAuthMethodKey = authenticationModes.some((method) => method.enabledConfigKey === key);

      // Only validate if this is an authentication method key
      if (isAuthMethodKey) {
        const canDisable = canDisableAuthMethod(key, authenticationModes, formattedConfig);

        if (!canDisable) {
          setToast({
            type: TOAST_TYPE.ERROR,
            title: "Cannot disable authentication",
            message:
              "At least one authentication method must remain enabled. Please enable another method before disabling this one.",
          });
          return;
        }
      }
    }

    notifyConfigurationSave(updateInstanceConfigurations({ [key]: value }));
  }

  const authenticationModes = useAuthenticationModes({
    config: formattedConfig,
    disabled: isSubmitting,
    updateConfig,
    resolvedTheme,
  });

  return (
    <PageWrapper
      header={{
        title: "Manage authentication modes for your instance",
        description: "Configure authentication modes for your team and restrict sign-ups to be invite only.",
      }}
    >
      {formattedConfig ? (
        <div className="space-y-3">
          <div className={cn("flex w-full items-center gap-14 rounded-sm")}>
            <div className="flex grow items-center gap-4">
              <div className="grow">
                <div className="pb-1 text-16 font-medium">Allow anyone to sign up even without an invite</div>
                <div className={cn("text-11 leading-5 font-regular text-tertiary")}>
                  Toggling this off will only let users sign up when they are invited.
                </div>
              </div>
            </div>
            <div className={`shrink-0 pr-4 ${isSubmitting && "opacity-70"}`}>
              <div className="flex items-center gap-4">
                <ToggleSwitch
                  value={Boolean(parseInt(enableSignUpConfig))}
                  onChange={() => {
                    if (parseInt(enableSignUpConfig)) {
                      updateConfig("ENABLE_SIGNUP", "0");
                    } else {
                      updateConfig("ENABLE_SIGNUP", "1");
                    }
                  }}
                  size="sm"
                  disabled={isSubmitting}
                />
              </div>
            </div>
          </div>
          <div className="text-lg pt-6 font-medium">Available authentication modes</div>
          {authenticationModes.map((method) => (
            <AuthenticationMethodCard
              key={method.key}
              name={method.name}
              description={method.description}
              icon={method.icon}
              config={method.config}
              disabled={isSubmitting}
              unavailable={method.unavailable}
            />
          ))}
        </div>
      ) : (
        <Loader className="space-y-10">
          <Loader.Item height="50px" width="75%" />
          <Loader.Item height="50px" width="75%" />
          <Loader.Item height="50px" width="40%" />
          <Loader.Item height="50px" width="40%" />
          <Loader.Item height="50px" width="20%" />
        </Loader>
      )}
    </PageWrapper>
  );
};

export const meta: Route.MetaFunction = () => [{ title: "Authentication Settings - Vector Web" }];

export default InstanceAuthenticationPage;
