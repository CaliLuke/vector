/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import type { ReactNode } from "react";
import type { useIssueEstimateOrder } from "@/hooks/use-issue-estimate-projection";

/** Never present a raw issue order as an accepted estimate order while metadata is unavailable. */
export function EstimateCatalogueBoundary({
  state,
  loading,
  children,
}: {
  state: Pick<ReturnType<typeof useIssueEstimateOrder>, "isError" | "isPending" | "retry">;
  loading?: ReactNode;
  children?: ReactNode;
}) {
  if (state.isError)
    return (
      <div role="alert">
        Work item estimates are unavailable.{" "}
        <button
          type="button"
          onClick={() => {
            void state.retry();
          }}
        >
          Retry
        </button>
      </div>
    );
  if (state.isPending) return loading ?? <output>Loading estimates…</output>;
  return children;
}
