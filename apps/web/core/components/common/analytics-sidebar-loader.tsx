/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Loader } from "@plane/ui";

export function AnalyticsSidebarLoader({ className }: { className?: string }) {
  return (
    <Loader className={className}>
      <div className="space-y-2">
        <Loader.Item height="15px" width="50%" />
        <Loader.Item height="15px" width="30%" />
      </div>
      <div className="mt-8 space-y-3">
        <Loader.Item height="30px" />
        <Loader.Item height="30px" />
        <Loader.Item height="30px" />
      </div>
    </Loader>
  );
}
