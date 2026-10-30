/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useLayoutEffect, useMemo, useRef, useState } from "react";

/** Keep clipboard feedback with the view that initiated the latest request. */
export function useClipboardCopy(scopeKey: unknown, resetAfterMs = 1000) {
  const scope = useMemo(() => ({ key: scopeKey }), [scopeKey]);
  const active = useRef<object | undefined>(undefined);
  const request = useRef(0);
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const [copiedScope, setCopiedScope] = useState<object | undefined>(undefined);

  useLayoutEffect(() => {
    active.current = scope;
    return () => {
      active.current = undefined;
      request.current += 1;
      clearTimeout(timer.current);
    };
  }, [scope]);

  const copy = async (write: () => Promise<void>): Promise<"copied" | "failed" | "stale"> => {
    if (active.current !== scope) return "stale";
    const current = ++request.current;
    clearTimeout(timer.current);
    setCopiedScope(undefined);
    const isCurrent = () => active.current === scope && request.current === current;
    try {
      await write();
      if (!isCurrent()) return "stale";
      setCopiedScope(scope);
      timer.current = setTimeout(() => setCopiedScope(undefined), resetAfterMs);
      return "copied";
    } catch {
      return isCurrent() ? "failed" : "stale";
    }
  };

  return { copied: copiedScope === scope, copy };
}
