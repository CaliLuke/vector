/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRouterState } from "@tanstack/react-router";
import { useNavigationProgress } from "@plane/hooks";
import { BProgress } from "@bprogress/core";

import "@bprogress/core/css";

export default function AppProgressBar(): null {
  const isLoading = useRouterState({ select: (state) => state.isLoading });
  useNavigationProgress(isLoading ? "loading" : "idle", BProgress);
  return null;
}
