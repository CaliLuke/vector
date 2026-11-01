/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useMemo, useSyncExternalStore } from "react";
import { createPortal } from "react-dom";
import { DEFAULT_PORTAL_ID } from "./constants";
import type { PortalWrapperProps } from "./types";

const owners = new WeakMap<HTMLElement, { count: number; created: boolean }>();
const serverSnapshot = () => null;

function createPortalStore(portalId: string) {
  let container: HTMLElement | null = null;
  return {
    getSnapshot: () => container,
    subscribe: (notify: () => void) => {
      const existing = document.getElementById(portalId);
      const target = existing ?? document.createElement("div");
      if (!existing) {
        target.id = portalId;
        target.setAttribute("data-portal", "true");
        document.body.append(target);
      }
      const ownership = owners.get(target) ?? { count: 0, created: !existing };
      owners.set(target, ownership);
      ownership.count++;
      container = target;
      notify();
      return () => {
        ownership.count--;
        // React removes portal children later in the unmount commit. Wait for
        // that removal, and allow a StrictMode remount to reacquire ownership.
        queueMicrotask(() => {
          if (ownership.created && ownership.count === 0 && !target.hasChildNodes()) target.remove();
        });
      };
    },
  };
}

/** Render into a shared container, creating and retiring it only while owned. */
export function PortalWrapper({
  children,
  portalId = DEFAULT_PORTAL_ID,
  className,
  onMount,
  onUnmount,
}: PortalWrapperProps) {
  const store = useMemo(() => createPortalStore(portalId), [portalId]);
  const container = useSyncExternalStore(store.subscribe, store.getSnapshot, serverSnapshot);
  useEffect(() => {
    if (!container) return undefined;
    onMount?.();
    return () => onUnmount?.();
  }, [container, onMount, onUnmount]);
  if (!container || !children) return null;
  return createPortal(className ? <div className={className}>{children}</div> : children, container);
}
