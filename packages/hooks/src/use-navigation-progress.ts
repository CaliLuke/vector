/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef } from "react";

interface ProgressConfig {
  showSpinner: boolean;
  minimum: number;
  speed: number;
  trickleSpeed: number;
  easing: string;
  trickle: boolean;
  delay: number;
}

interface ProgressController {
  configure(options: Omit<ProgressConfig, "delay">): unknown;
  render(fromStart?: boolean): unknown;
  start(): unknown;
  done(): unknown;
  isStarted(): boolean;
}

const PROGRESS_CONFIG: Readonly<ProgressConfig> = {
  showSpinner: false,
  minimum: 0.1,
  speed: 400,
  trickleSpeed: 800,
  easing: "ease",
  trickle: true,
  delay: 0,
};

/**
 * Configure a progress controller and track a router's idle/loading/submitting state.
 */
export function useNavigationProgress(navigationState: string, progress: ProgressController): void {
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const startedRef = useRef(false);

  useEffect(() => {
    progress.configure({
      showSpinner: PROGRESS_CONFIG.showSpinner,
      minimum: PROGRESS_CONFIG.minimum,
      speed: PROGRESS_CONFIG.speed,
      trickleSpeed: PROGRESS_CONFIG.trickleSpeed,
      easing: PROGRESS_CONFIG.easing,
      trickle: PROGRESS_CONFIG.trickle,
    });
    progress.render(true);

    return () => {
      if (progress.isStarted()) progress.done();
    };
  }, [progress]);

  useEffect(() => {
    if (navigationState === "idle") {
      if (timerRef.current !== null) {
        clearTimeout(timerRef.current);
        timerRef.current = null;
      }

      if (startedRef.current) {
        progress.done();
        startedRef.current = false;
      }
    } else if (timerRef.current === null && !startedRef.current) {
      timerRef.current = setTimeout((): void => {
        if (!progress.isStarted()) {
          progress.start();
          startedRef.current = true;
        }
        timerRef.current = null;
      }, PROGRESS_CONFIG.delay);
    }

    return () => {
      if (timerRef.current !== null) {
        clearTimeout(timerRef.current);
        timerRef.current = null;
      }
    };
  }, [navigationState, progress]);
}
