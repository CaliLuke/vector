/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { useEffect, useRef } from "react";

import { useTheme } from "next-themes";
import { FALLBACK_LANGUAGE, SUPPORTED_LANGUAGES, setLanguage } from "@plane/i18n";
import type { TLanguage } from "@plane/i18n";
// helpers
import { applyCustomTheme, clearCustomTheme } from "@plane/utils/theme";
// hooks
import { useCurrentUser } from "@/hooks/use-current-user";
import { useCurrentUserProfile } from "@/hooks/use-current-user-profile";

let languageRevision = 0;
let appliedLanguageRevision = 0;
let activeLanguage: TLanguage = FALLBACK_LANGUAGE;
let languageRunner: Promise<void> | undefined;

function applyNextLanguage(): Promise<void> {
  if (appliedLanguageRevision === languageRevision) return Promise.resolve();
  const revision = languageRevision;
  return setLanguage(activeLanguage)
    .catch((error: unknown) => {
      console.error("Failed to apply the current account language", error);
    })
    .then(() => {
      appliedLanguageRevision = revision;
      return applyNextLanguage();
    });
}

function runLatestLanguage() {
  if (languageRunner) return;
  languageRunner = applyNextLanguage().finally(() => {
    languageRunner = undefined;
    if (appliedLanguageRevision !== languageRevision) runLatestLanguage();
  });
}

function applyCurrentLanguage(language: TLanguage) {
  activeLanguage = language;
  languageRevision++;
  runLatestLanguage();
}

type ProfileEffectsProps = {
  children: ReactNode;
};

function useProfileTheme(
  currentUser: ReturnType<typeof useCurrentUser>["data"],
  profile: ReturnType<typeof useCurrentUserProfile>["data"],
  setTheme: (theme: string) => void
) {
  const hasInitializedThemeRef = useRef(false);
  const currentUserIdRef = useRef<string | undefined>(undefined);
  const previousThemeRef = useRef<string | undefined>(undefined);

  useEffect(() => {
    if (currentUser?.id !== currentUserIdRef.current) {
      if (previousThemeRef.current === "custom") clearCustomTheme();
      hasInitializedThemeRef.current = false;
      previousThemeRef.current = undefined;
      currentUserIdRef.current = currentUser?.id;
    }

    if (!currentUser || profile?.user !== currentUser.id || !profile.theme?.theme || hasInitializedThemeRef.current)
      return;
    setTheme(profile.theme.theme);
    hasInitializedThemeRef.current = true;
  }, [currentUser, profile, setTheme]);

  useEffect(() => {
    if (!currentUser || profile?.user !== currentUser.id || !profile.theme?.theme) {
      clearCustomTheme();
      previousThemeRef.current = undefined;
      return;
    }

    const theme = profile.theme.theme;
    const previousTheme = previousThemeRef.current;
    const themeData = profile.theme;
    if (theme === "custom" && themeData.primary && themeData.background && themeData.darkPalette !== undefined) {
      applyCustomTheme(themeData.primary, themeData.background, themeData.darkPalette ? "dark" : "light");
    } else if (previousTheme === "custom" && theme !== "custom") {
      clearCustomTheme();
    }
    previousThemeRef.current = theme;
  }, [currentUser, profile]);
}

function getCurrentProfileLanguage(
  currentUser: ReturnType<typeof useCurrentUser>["data"],
  profile: ReturnType<typeof useCurrentUserProfile>["data"]
): TLanguage {
  if (!currentUser || profile?.user !== currentUser.id) return FALLBACK_LANGUAGE;
  return SUPPORTED_LANGUAGES.find(({ value }) => value === profile.language)?.value ?? FALLBACK_LANGUAGE;
}

function ProfileEffects(props: ProfileEffectsProps) {
  const { children } = props;
  // theme
  const { setTheme } = useTheme();
  const { data: currentUser } = useCurrentUser();
  const { data: userProfile } = useCurrentUserProfile();
  useProfileTheme(currentUser, userProfile, setTheme);

  useEffect(() => {
    applyCurrentLanguage(getCurrentProfileLanguage(currentUser, userProfile));
  }, [currentUser, userProfile]);

  useEffect(
    () => () => {
      applyCurrentLanguage(FALLBACK_LANGUAGE);
      clearCustomTheme();
    },
    []
  );

  return <>{children}</>;
}

export default ProfileEffects;
