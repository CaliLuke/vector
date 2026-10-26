/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useCallback, useLayoutEffect, useMemo, useRef } from "react";

/** A completion belongs to this committed scope, including route round trips and unmounts. */
export function useCommittedScope(key: string) {
  const token = useMemo(() => ({ key }), [key]);
  const active = useRef<object | undefined>(undefined);
  useLayoutEffect(() => {
    active.current = token;
    return () => {
      active.current = undefined;
    };
  }, [token]);
  return useCallback(() => active.current === token, [token]);
}
