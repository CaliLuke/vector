/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { JSX, ReactNode, MutableRefObject } from "react";
import React, { useState, useRef, useEffect } from "react";
import { cn } from "@plane/utils/common";
import { runIdleTask } from "@/lib/idle-task";

type Props = {
  defaultHeight?: string;
  verticalOffset?: number;
  horizontalOffset?: number;
  root?: MutableRefObject<HTMLElement | null>;
  children: ReactNode;
  as?: keyof JSX.IntrinsicElements;
  classNames?: string;
  placeholderChildren?: ReactNode;
  defaultValue?: boolean;
  shouldRecordHeights?: boolean;
  useIdleTime?: boolean;
  forceRender?: boolean;
};

function RenderIfVisible(props: Props) {
  const {
    defaultHeight = "300px",
    as = "div",
    children,
    classNames = "",
    shouldRecordHeights = true,
    //placeholder children
    placeholderChildren = null, //placeholder children
    forceRender = false,
  } = props;
  const [placeholderHeight, setPlaceholderHeight] = useState(defaultHeight);
  const { intersectionRef, shouldVisible } = useObservedVisibility(props);
  const heightIdleTaskRef = useRef<ReturnType<typeof runIdleTask> | null>(null);

  const isVisible = shouldVisible || forceRender;

  //Set height after render
  useEffect(() => {
    if (intersectionRef.current && isVisible && shouldRecordHeights) {
      heightIdleTaskRef.current?.cancel();
      heightIdleTaskRef.current = runIdleTask(() => {
        if (intersectionRef.current) setPlaceholderHeight(`${intersectionRef.current.offsetHeight}px`);
      });
    }
    return () => {
      heightIdleTaskRef.current?.cancel();
      heightIdleTaskRef.current = null;
    };
  }, [isVisible, intersectionRef, shouldRecordHeights]);

  const child = isVisible ? <>{children}</> : placeholderChildren;
  const style = isVisible || !shouldRecordHeights ? {} : { height: placeholderHeight, width: "100%" };
  const className = isVisible || placeholderChildren ? classNames : cn(classNames, "bg-layer-1");

  return React.createElement(as, { ref: intersectionRef, style, className }, child);
}

function useObservedVisibility({
  root,
  verticalOffset = 50,
  horizontalOffset = 0,
  defaultValue = false,
  useIdleTime = false,
}: Props) {
  const [shouldVisible, setShouldVisible] = useState(defaultValue);
  const intersectionRef = useRef<HTMLElement | null>(null);
  const visibilityIdleTaskRef = useRef<ReturnType<typeof runIdleTask> | null>(null);
  // Set visibility with intersection observer
  useEffect(() => {
    const target = intersectionRef.current;
    if (target) {
      let active = true;
      const observer = new IntersectionObserver(
        (entries) => {
          const latest = entries.at(-1);
          if (!active || !latest) return;
          if (typeof window !== "undefined" && useIdleTime) {
            visibilityIdleTaskRef.current?.cancel();
            visibilityIdleTaskRef.current = runIdleTask(() => setShouldVisible(latest.isIntersecting));
          } else {
            setShouldVisible(latest.isIntersecting);
          }
        },
        {
          root: root?.current,
          rootMargin: `${verticalOffset}% ${horizontalOffset}% ${verticalOffset}% ${horizontalOffset}%`,
        }
      );
      observer.observe(target);
      return () => {
        visibilityIdleTaskRef.current?.cancel();
        visibilityIdleTaskRef.current = null;
        active = false;
        observer.disconnect();
      };
    }

    return undefined;
  }, [intersectionRef, root, verticalOffset, horizontalOffset, useIdleTime]);

  return { intersectionRef, shouldVisible };
}

export default RenderIfVisible;
