/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useMemo } from "react";

import { useTheme } from "next-themes";
// plane imports
import type { I_THEME_OPTION } from "@plane/constants";
import { THEME_OPTIONS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { setPromiseToast } from "@plane/propel/toast";
import { applyCustomTheme, clearCustomTheme } from "@plane/utils/theme";
// components
import { CustomThemeSelector } from "@/components/core/theme/custom-theme-selector";
import { ThemeSwitch } from "@/components/core/theme/theme-switch";
import { SettingsControlItem } from "@/components/settings/control-item";
// hooks
import { useCurrentUserProfile, useUpdateUserTheme } from "@/hooks/use-current-user-profile";

export const ThemeSwitcher = function ThemeSwitcher(props: {
  option: {
    id: string;
    title: string;
    description: string;
  };
}) {
  // store hooks
  const { data: userProfile } = useCurrentUserProfile();
  const { mutateAsync: updateUserTheme } = useUpdateUserTheme();
  // theme
  const { setTheme } = useTheme();
  // translation
  const { t } = useTranslation();
  // derived values
  const currentTheme = useMemo(() => {
    // oxlint-disable-next-line no-shadow
    const userThemeOption = THEME_OPTIONS.find((t) => t.value === userProfile?.theme?.theme);
    return userThemeOption || null;
  }, [userProfile?.theme?.theme]);

  const handleThemeChange = useCallback(
    async (themeOption: I_THEME_OPTION) => {
      try {
        const updatePromise = updateUserTheme({ theme: themeOption.value });
        setPromiseToast(updatePromise, {
          loading: "Updating theme...",
          success: {
            title: "Theme updated",
            message: () => "Your theme has been applied.",
          },
          error: {
            title: "Error!",
            message: () => "Failed to update theme. Please try again.",
          },
        });
        await updatePromise;
        setTheme(themeOption.value);

        // If switching to custom theme and user has saved custom colors, apply them immediately
        if (
          themeOption.value === "custom" &&
          userProfile?.theme?.primary &&
          userProfile?.theme?.background &&
          userProfile?.theme?.darkPalette !== undefined
        ) {
          applyCustomTheme(
            userProfile.theme.primary,
            userProfile.theme.background,
            userProfile.theme.darkPalette ? "dark" : "light"
          );
        } else {
          clearCustomTheme();
        }
      } catch (error) {
        console.error("Error updating theme:", error);
      }
    },
    [setTheme, updateUserTheme, userProfile]
  );

  if (!userProfile) return null;

  return (
    <>
      <SettingsControlItem
        title={t(props.option.title)}
        description={t(props.option.description)}
        control={
          <ThemeSwitch
            value={currentTheme}
            onChange={(themeOption) => {
              void handleThemeChange(themeOption);
            }}
          />
        }
      />
      {userProfile.theme?.theme === "custom" && <CustomThemeSelector />}
    </>
  );
};
