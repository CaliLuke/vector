/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useEffect, useMemo, useRef, useSyncExternalStore } from "react";

type TArgs = { elementId: string; pathname: string; scrollDelay?: number };
type TReturnType = { isHashMatch: boolean; hashIds: string[]; scrollToElement: () => boolean };

function subscribeToHash(listener: () => void) {
  window.addEventListener("hashchange", listener);
  return () => window.removeEventListener("hashchange", listener);
}
const getServerSnapshot = () => "";

function readHashIds(location: string) {
  if (!location) return [];
  return new URL(location).hash
    .slice(1)
    .split(/[,\s|;]+/)
    .filter(Boolean);
}

/** Observe the route's hash and cancel deferred scrolling when its target changes. */
export function useHashScroll({ elementId, pathname, scrollDelay = 200 }: TArgs): TReturnType {
  const getSnapshot = useCallback(() => {
    const location = new URL(pathname, window.location.origin);
    location.hash = window.location.hash;
    return location.href;
  }, [pathname]);
  const location = useSyncExternalStore(subscribeToHash, getSnapshot, getServerSnapshot);
  const hashIds = useMemo(() => (elementId ? readHashIds(location) : []), [elementId, location]);
  const pendingScroll = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const cancelPendingScroll = useCallback(() => {
    clearTimeout(pendingScroll.current);
    pendingScroll.current = undefined;
  }, []);
  const scrollToElement = useCallback(() => {
    cancelPendingScroll();
    const element = document.getElementById(elementId);
    if (!element || window.location.pathname !== pathname) return false;
    pendingScroll.current = setTimeout(() => {
      pendingScroll.current = undefined;
      if (!element.isConnected || window.location.pathname !== pathname) return;
      try {
        element.scrollIntoView({ behavior: "smooth", block: "nearest" });
      } catch (error) {
        console.warn("Hash scroll error:", error);
      }
    }, scrollDelay);
    return true;
  }, [elementId, pathname, scrollDelay, cancelPendingScroll]);
  useEffect(() => {
    if (hashIds.includes(elementId)) scrollToElement();
    return cancelPendingScroll;
  }, [hashIds, elementId, scrollToElement, cancelPendingScroll]);
  return { hashIds, isHashMatch: hashIds.includes(elementId), scrollToElement };
}
