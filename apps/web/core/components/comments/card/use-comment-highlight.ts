/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useState } from "react";
import { useHashScroll } from "@plane/hooks";

export function useCommentHighlight(elementId: string, pathname: string) {
  const { isHashMatch } = useHashScroll({ elementId, pathname });
  const [highlight, setHighlight] = useState({ isHashMatch, expired: false });
  if (highlight.isHashMatch !== isHashMatch) setHighlight({ isHashMatch, expired: false });

  useEffect(() => {
    if (!isHashMatch) return undefined;
    const timeout = setTimeout(() => setHighlight((current) => ({ ...current, expired: true })), 8000);
    return () => clearTimeout(timeout);
  }, [isHashMatch]);

  return isHashMatch && !highlight.expired ? "border-accent-strong" : "";
}
