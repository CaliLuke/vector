/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTheme } from "next-themes";
import { Moon, Sun } from "lucide-react";

export function NavbarTheme() {
  const { setTheme, resolvedTheme } = useTheme();
  return (
    <button
      type="button"
      aria-label="Toggle color theme"
      onClick={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")}
      className="relative grid size-7 place-items-center rounded-sm bg-layer-transparent text-primary hover:bg-layer-transparent-hover"
    >
      <Moon className="size-3.5 shrink-0 dark:hidden" />
      <Sun className="hidden size-3.5 shrink-0 dark:block" />
    </button>
  );
}
