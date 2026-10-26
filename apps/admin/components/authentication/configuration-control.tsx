/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link } from "react-router";
import { Settings2 } from "lucide-react";
import { getButtonStyling } from "@plane/propel/button";
import type { TInstanceAuthenticationMethodKeys } from "@plane/types";
import { ToggleSwitch } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { oauthSettings } from "./oauth-settings";
import type { OAuthProvider, OAuthValues } from "./oauth-settings";

type Props = {
  config: OAuthValues | undefined;
  configKey: TInstanceAuthenticationMethodKeys;
  provider?: OAuthProvider;
  disabled: boolean;
  updateConfig: (key: TInstanceAuthenticationMethodKeys, value: string) => void;
};
export function ConfigurationControl({ config, configKey, provider, disabled, updateConfig }: Props) {
  const settings = provider ? oauthSettings[provider] : undefined;
  const configured =
    settings?.fields.filter((field) => field.required).every((field) => Boolean(config?.[field.key])) ?? true;
  const enabled = config?.[configKey] === "1";
  if (provider && !configured)
    return (
      <Link to={`/authentication/${provider}`} className={cn(getButtonStyling("secondary", "base"), "text-tertiary")}>
        <Settings2 className="h-4 w-4 p-0.5 text-tertiary" />
        Configure
      </Link>
    );
  return (
    <div className="flex items-center gap-4">
      {provider && (
        <Link to={`/authentication/${provider}`} className={cn(getButtonStyling("link", "base"), "font-medium")}>
          Edit
        </Link>
      )}
      <ToggleSwitch
        value={enabled}
        onChange={() => updateConfig(configKey, enabled ? "0" : "1")}
        size="sm"
        disabled={disabled}
      />
    </div>
  );
}
