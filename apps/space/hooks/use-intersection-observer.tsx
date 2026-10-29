/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { RefObject } from "react";
import { useEffect } from "react";

export const useIntersectionObserver = (
  containerRef: RefObject<HTMLDivElement | null>,
  elementRef: HTMLElement | null,
  callback: (() => void) | undefined,
  rootMargin?: string
) => {
  useEffect(() => {
    if (elementRef) {
      const observer = new IntersectionObserver(
        (entries) => {
          if (entries[entries.length - 1].isIntersecting) {
            if (callback) {
              callback();
            }
          }
        },
        {
          root: containerRef?.current,
          rootMargin,
        }
      );
      observer.observe(elementRef);
      return () => {
        observer.disconnect();
      };
    }
  }, [rootMargin, callback, elementRef, containerRef]);
};
