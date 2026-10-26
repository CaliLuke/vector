/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { createRoot } from "react-dom/client";

export function renderNativeDragPreview(container: HTMLElement, preview: ReactNode) {
  const root = createRoot(container);
  root.render(preview);
  return () => root.unmount();
}
