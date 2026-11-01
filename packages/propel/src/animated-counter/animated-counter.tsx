/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState, useEffect } from "react";
import { cn } from "@plane/utils/common";

export interface AnimatedCounterProps {
  count: number;
  className?: string;
  size?: "sm" | "md" | "lg";
}

const sizeClasses = {
  sm: "text-11",
  md: "text-13",
  lg: "text-14",
};

export function AnimatedCounter({ count, className, size = "md" }: AnimatedCounterProps) {
  const [transition, setTransition] = useState({ count, previous: count, key: 0 });
  const [completedKey, setCompletedKey] = useState(0);
  if (transition.count !== count) {
    setTransition({ count, previous: transition.count, key: transition.key + 1 });
  }
  const { count: displayCount, previous: prevCount, key: animationKey } = transition;
  const isAnimating = animationKey !== completedKey;
  const direction = displayCount > prevCount ? "up" : "down";

  useEffect(() => {
    if (animationKey === 0) return undefined;
    const timer = setTimeout(() => setCompletedKey(animationKey), 250);
    return () => clearTimeout(timer);
  }, [animationKey]);

  const sizeClass = sizeClasses[size];

  return (
    <div className={cn("relative inline-flex min-w-2 items-center justify-center overflow-hidden", sizeClass)}>
      {/* Previous number sliding out */}
      {isAnimating && (
        <span
          key={`prev-${animationKey}`}
          className={cn(
            "absolute inset-0 flex items-center justify-center font-medium",
            "animate-slide-out",
            direction === "up" && "[--slide-out-dir:-100%]",
            direction === "down" && "[--slide-out-dir:100%]",
            sizeClass,
            {
              "animate-fade-out animate-slide-out": isAnimating && direction === "up",
              "animate-fade-out animate-slide-out-down": isAnimating && direction === "down",
            }
          )}
        >
          {prevCount}
        </span>
      )}

      {/* New number sliding in */}
      <span
        key={`current-${animationKey}`}
        className={cn(
          "flex items-center justify-center font-medium",
          !isAnimating && "opacity-100",
          sizeClass,
          {
            "animate-slide-in-from-bottom": isAnimating && direction === "up",
            "animate-slide-in-from-top": isAnimating && direction === "down",
          },
          className
        )}
      >
        {displayCount}
      </span>
    </div>
  );
}
