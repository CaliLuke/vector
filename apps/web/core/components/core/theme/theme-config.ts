/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export type CustomThemeConfig = { theme: "custom"; primary: string; background: string; darkPalette: boolean };

export const isThemeColor = (value: unknown): value is string =>
  typeof value === "string" && /^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$/.test(value);

export function parseThemeConfig(text: string): CustomThemeConfig {
  const config: unknown = JSON.parse(text);
  if (!config || typeof config !== "object") throw new Error("Expected a theme configuration object");
  if (!("primary" in config) || !isThemeColor(config.primary)) throw new Error("Invalid brand color hex format");
  if (!("background" in config) || !isThemeColor(config.background))
    throw new Error("Invalid neutral color hex format");
  const darkPalette = "darkPalette" in config ? config.darkPalette : false;
  if (typeof darkPalette !== "boolean") throw new Error("Invalid theme mode. Must be a boolean");
  return { theme: "custom", primary: config.primary, background: config.background, darkPalette };
}
