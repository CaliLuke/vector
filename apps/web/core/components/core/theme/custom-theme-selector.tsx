/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useForm, useStore } from "@tanstack/react-form";
import { useTheme } from "next-themes";
import { isThemeColor, type CustomThemeConfig } from "./theme-config";
// plane imports
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { applyCustomTheme } from "@plane/utils/theme";
// components
import { ProfileSettingsHeading } from "@/components/settings/profile/heading";
// hooks
import { useCurrentUserProfile, useUpdateUserTheme } from "@/hooks/use-current-user-profile";
// local imports
import { CustomThemeColorInput } from "./color-inputs";
import { CustomThemeDownloadConfigButton } from "./download-config-button";
import { CustomThemeImportConfigButton } from "./import-config-button";
import { CustomThemeModeSelector } from "./theme-mode-selector";

export const CustomThemeSelector = function CustomThemeSelector() {
  // store hooks
  const { data: userProfile } = useCurrentUserProfile();
  const { mutateAsync: updateUserTheme } = useUpdateUserTheme();
  // translation
  const { t } = useTranslation();

  const { setTheme } = useTheme();
  const savedTheme: CustomThemeConfig = {
    theme: "custom",
    primary: userProfile?.theme?.primary || "#3f76ff",
    background: userProfile?.theme?.background || "#1a1a1a",
    darkPalette: userProfile?.theme?.darkPalette ?? false,
  };
  const form = useForm({
    defaultValues: savedTheme,
    onSubmit: async ({ value, formApi }) => {
      try {
        await updateUserTheme(value);
        setTheme("custom");
        applyCustomTheme(value.primary, value.background, value.darkPalette ? "dark" : "light");
        formApi.reset(value);
        setToast({ type: TOAST_TYPE.SUCCESS, title: t("success"), message: "Theme updated." });
      } catch (error) {
        setToast({ type: TOAST_TYPE.ERROR, title: t("error"), message: t("failed_to_update_the_theme") });
        throw error;
      }
    },
  });
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const values = useStore(form.store, (state) => state.values);
  const importTheme = async (value: CustomThemeConfig) => {
    form.setFieldValue("primary", value.primary);
    form.setFieldValue("background", value.background);
    form.setFieldValue("darkPalette", value.darkPalette);
    await form.handleSubmit().catch(() => {
      // The failed theme update is already reported with a toast.
    });
  };

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        void form.handleSubmit().catch(() => {
          // The failed theme update is already reported with a toast.
        });
      }}
      className="rounded-lg border border-subtle bg-layer-1 px-4 py-3"
    >
      <div className="space-y-5">
        <ProfileSettingsHeading
          title={t("customize_your_theme")}
          control={<CustomThemeImportConfigButton onImport={importTheme} disabled={isSubmitting} />}
        />
        <form.Field name="darkPalette">
          {(field) => (
            <CustomThemeModeSelector value={field.state.value} onChange={field.handleChange} disabled={isSubmitting} />
          )}
        </form.Field>
        {/* Color Inputs */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {(["background", "primary"] as const).map((name) => (
            <form.Field
              key={name}
              name={name}
              validators={{ onChange: ({ value }) => (isThemeColor(value) ? undefined : "Enter a valid hex code") }}
            >
              {(field) => (
                <CustomThemeColorInput
                  name={name}
                  label={name === "background" ? "Neutral color" : "Brand color"}
                  value={field.state.value}
                  onChange={field.handleChange}
                  error={field.state.meta.errors.join(" ")}
                  disabled={isSubmitting}
                />
              )}
            </form.Field>
          ))}
        </div>
      </div>
      <div className="mt-5 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        {/* Save Theme Button */}
        <Button variant="primary" size="lg" type="submit" loading={isSubmitting}>
          {isSubmitting ? t("common.saving") : t("set_theme")}
        </Button>
        {/* Import/Export Section */}
        <CustomThemeDownloadConfigButton value={values} />
      </div>
    </form>
  );
};
