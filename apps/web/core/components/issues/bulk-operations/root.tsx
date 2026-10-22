/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { Button } from "@plane/propel/button";
import { cn } from "@plane/utils";
// hooks
import { useMultipleSelectStore } from "@/hooks/store/use-multiple-select-store";
import type { TSelectionHelper } from "@/hooks/use-multiple-select";

type Props = {
  className?: string;
  selectionHelpers: TSelectionHelper;
};

export const IssueBulkOperationsRoot = observer(function IssueBulkOperationsRoot(props: Props) {
  const { className, selectionHelpers } = props;
  const { isSelectionActive } = useMultipleSelectStore();

  if (!isSelectionActive || selectionHelpers.isSelectionDisabled) return null;

  return (
    <div
      className={cn(
        "sticky bottom-0 z-[2] flex items-center justify-between gap-3 border-t border-subtle bg-surface-1 p-3",
        className
      )}
    >
      <p className="text-13 text-secondary">
        Bulk editing is not available in this build. Open a work item to edit it.
      </p>
      <Button variant="secondary" size="sm" onClick={selectionHelpers.handleClearSelection}>
        Clear selection
      </Button>
    </div>
  );
});
