/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { RefObject } from "react";
import { useEffect, useEffectEvent } from "react";

export const useIntersectionObserver = (
  containerRef: RefObject<HTMLDivElement | null>,
  element: HTMLElement | null,
  callback: (() => void) | undefined,
  rootMargin?: string
) => {
  const onIntersect = useEffectEvent(() => callback?.());
  useEffect(() => {
    if (!element) return undefined;
    let active = true;
    const observer = new IntersectionObserver(
      (entries) => {
        if (active && entries.at(-1)?.isIntersecting) onIntersect();
      },
      { root: containerRef.current, rootMargin }
    );
    observer.observe(element);
    return () => {
      active = false;
      observer.disconnect();
    };
  }, [containerRef, element, rootMargin]);
};
