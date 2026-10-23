/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useNavigationProgress } from "@plane/hooks";
import { BProgress } from "@bprogress/core";
import { useNavigation } from "react-router";
import "@bprogress/core/css";

export function AppProgressBar(): null {
  const navigation = useNavigation();
  useNavigationProgress(navigation.state, BProgress);
  return null;
}
