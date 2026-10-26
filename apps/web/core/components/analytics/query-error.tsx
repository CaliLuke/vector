/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";

type Props = {
  onRetry: () => void;
};

export function AnalyticsQueryError({ onRetry }: Props) {
  return (
    <div className="text-sm text-error flex items-center gap-2" role="alert">
      <span>Analytics could not be loaded.</span>
      <button className="underline" onClick={onRetry} type="button">
        Retry
      </button>
    </div>
  );
}

export function AnalyticsAccessDenied() {
  return <output className="text-sm text-secondary">You do not have access to these analytics.</output>;
}
