/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect } from "react";
import type { RefObject } from "react";

export function useOverviewOutsideClick(
  ref: RefObject<HTMLElement | null>,
  callback: () => void,
  targetId: string,
  {
    allowInsidePrevention = false,
    excludePreventionElementIds,
  }: {
    allowInsidePrevention?: boolean;
    excludePreventionElementIds?: string[];
  } = {}
) {
  useEffect(() => {
    const handleClick = (event: MouseEvent) => {
      const target = event.target;
      const panel = ref.current;
      if (!(target instanceof HTMLElement) || !panel || panel.contains(target)) return;
      const prevention = target.closest("[data-prevent-outside-click]");
      if (prevention) {
        const excluded = prevention.id && excludePreventionElementIds?.includes(prevention.id);
        if (!excluded && !(allowInsidePrevention && prevention.contains(panel))) return;
      }
      let ancestor: HTMLElement | null = target;
      while (ancestor) {
        if (ancestor.id === targetId) return;
        ancestor = ancestor.parentElement;
      }
      if (target.closest("[data-delay-outside-click]")) setTimeout(() => callback(), 0);
      else callback();
    };
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [ref, callback, targetId, allowInsidePrevention, excludePreventionElementIds]);
}
