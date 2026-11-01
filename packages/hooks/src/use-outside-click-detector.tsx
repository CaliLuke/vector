/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type React from "react";
import { useEffect } from "react";

export const useOutsideClickDetector = (
  ref: React.RefObject<HTMLElement | null>,
  callback: () => void,
  useCapture = false
) => {
  useEffect(() => {
    const handleClick = (event: MouseEvent) => {
      const target = event.target;
      if (!ref.current || !(target instanceof Node) || ref.current.contains(target)) return;
      const element = target instanceof Element ? target : target.parentElement;
      if (element?.closest("[data-prevent-outside-click]")) return;
      callback();
    };
    document.addEventListener("mousedown", handleClick, useCapture);
    return () => document.removeEventListener("mousedown", handleClick, useCapture);
  }, [ref, callback, useCapture]);
};
