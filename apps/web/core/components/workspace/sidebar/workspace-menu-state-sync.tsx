/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect } from "react";

type WorkspaceMenuStateSyncProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function WorkspaceMenuStateSync({ open, onOpenChange }: WorkspaceMenuStateSyncProps) {
  useEffect(() => {
    onOpenChange(open);
  }, [open, onOpenChange]);

  return null;
}
