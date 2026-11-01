/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useSyncExternalStore } from "react";
import type { ReactNode } from "react";
import { createPortal } from "react-dom";

const subscribe = () => () => {};
const clientSnapshot = () => true;
const serverSnapshot = () => false;

export function DropdownPortal({
  children,
  container,
  wrap = false,
}: {
  children: ReactNode;
  container?: Element | null;
  wrap?: boolean;
}) {
  const hydrated = useSyncExternalStore(subscribe, clientSnapshot, serverSnapshot);
  if (!hydrated) return null;
  return createPortal(wrap ? <div data-radix-portal="">{children}</div> : children, container ?? document.body);
}
