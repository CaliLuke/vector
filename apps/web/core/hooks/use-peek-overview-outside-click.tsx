/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type React from "react";
import { useOverviewOutsideClick } from "./use-overview-outside-click";

export default function usePeekOverviewOutsideClickDetector(
  ref: React.RefObject<HTMLElement | null>,
  callback: () => void,
  issueId: string,
  excludePreventionElementIds?: string[]
) {
  useOverviewOutsideClick(ref, callback, `issue-${issueId}`, {
    allowInsidePrevention: true,
    excludePreventionElementIds,
  });
}
