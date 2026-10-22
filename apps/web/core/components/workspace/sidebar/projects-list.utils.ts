/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export const findSidebarScrollViewport = (container: HTMLElement | null): HTMLElement | null =>
  container?.closest<HTMLElement>('[data-slot="scroll-area-viewport"]') ?? null;
