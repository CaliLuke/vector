/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { Button } from "@plane/propel/button";
export function CatalogueReadFailure({ label, retry }: { label: string; retry: () => Promise<unknown> }) {
  return (
    <div role="alert" className="flex items-center gap-3 p-5">
      <p>Unable to load {label}.</p>
      <Button onClick={() => void retry().catch((error) => console.warn(`The ${label} retry failed.`, error))}>
        Retry
      </Button>
    </div>
  );
}
