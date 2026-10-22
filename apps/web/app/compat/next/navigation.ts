/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useEffect, useLayoutEffect, useMemo, useRef } from "react";
import { useLocation, useNavigate, useParams as useParamsRR, useSearchParams as useSearchParamsRR } from "react-router";
import { ensureTrailingSlash } from "./helper";

type TNavigationRequest = {
  to: string | number;
  options?: { replace?: boolean };
};

const useIsomorphicLayoutEffect = typeof window === "undefined" ? useEffect : useLayoutEffect;

export function useRouter() {
  const navigate = useNavigate();
  const navigateRef = useRef(navigate);
  const isCommittedRef = useRef(false);
  const pendingNavigationsRef = useRef<TNavigationRequest[]>([]);
  const timersRef = useRef(new Set<ReturnType<typeof setTimeout>>());
  navigateRef.current = navigate;

  const scheduleCommittedNavigation = useCallback((request: TNavigationRequest, cancelOnUnmount = false) => {
    const timer = setTimeout(() => {
      timersRef.current.delete(timer);
      if (cancelOnUnmount && !isCommittedRef.current) return;
      pendingNavigationsRef.current = pendingNavigationsRef.current.filter((pending) => pending !== request);
      if (typeof request.to === "number") navigateRef.current(request.to);
      else navigateRef.current(request.to, request.options);
    }, 0);
    // An accepted event navigation must survive its modal or menu closing.
    // Only redirects queued before the first commit belong to this component.
    if (cancelOnUnmount) timersRef.current.add(timer);
  }, []);

  useIsomorphicLayoutEffect(() => {
    isCommittedRef.current = true;
    for (const request of pendingNavigationsRef.current) scheduleCommittedNavigation(request, true);

    return () => {
      isCommittedRef.current = false;
      for (const timer of timersRef.current) clearTimeout(timer);
      timersRef.current.clear();
    };
  }, [scheduleCommittedNavigation]);

  const scheduleNavigation = useCallback(
    (request: TNavigationRequest) => {
      if (!isCommittedRef.current) {
        const pending = pendingNavigationsRef.current;
        const previous = pending[pending.length - 1];
        if (previous?.to !== request.to || previous.options?.replace !== request.options?.replace)
          pending.push(request);
        return;
      }

      scheduleCommittedNavigation(request);
    },
    [scheduleCommittedNavigation]
  );

  return useMemo(
    () => ({
      push: (to: string) => {
        scheduleNavigation({ to: ensureTrailingSlash(to) });
      },
      replace: (to: string) => {
        scheduleNavigation({ to: ensureTrailingSlash(to), options: { replace: true } });
      },
      back: () => {
        scheduleNavigation({ to: -1 });
      },
      forward: () => {
        scheduleNavigation({ to: 1 });
      },
      refresh: () => {
        location.reload();
      },
      prefetch: async (_to: string) => {
        // no-op in this shim
      },
    }),
    [scheduleNavigation]
  );
}

export function usePathname(): string {
  const { pathname } = useLocation();
  return pathname;
}

export function useSearchParams(): URLSearchParams {
  const [searchParams] = useSearchParamsRR();
  return searchParams;
}

export function useParams() {
  return useParamsRR();
}
