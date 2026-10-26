/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import type { ReactNode } from "react";
import type { useEstimateStatistics } from "@/hooks/use-estimate-statistics";

export function EstimateStatisticsBoundary({
  state,
  children,
}: {
  state: ReturnType<typeof useEstimateStatistics>["statistics"];
  children: ReactNode;
}) {
  if (state.isError)
    return (
      <div role="alert">
        Statistics could not be refreshed.{" "}
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
  if (state.isPending) return <output>Updating statistics…</output>;
  return children;
}
