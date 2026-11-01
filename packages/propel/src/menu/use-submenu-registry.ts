/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useCallback, useRef } from "react";

/** Parent menus own the close callbacks of their currently mounted submenus. */
export function useSubmenuRegistry() {
  const closers = useRef(new Set<() => void>());
  const closeAllSubmenus = useCallback(() => {
    closers.current.forEach((close) => close());
  }, []);
  const registerSubmenu = useCallback((close: () => void) => {
    closers.current.add(close);
    return () => {
      closers.current.delete(close);
    };
  }, []);
  return { closeAllSubmenus, registerSubmenu };
}
