/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { copyUrlToClipboard } from "@plane/utils/string";
import { useCommittedScope } from "@/hooks/use-committed-scope";

export function useViewLinkCopy(path: string) {
  const isCurrent = useCommittedScope(path);
  return async () => {
    try {
      await copyUrlToClipboard(path);
      if (!isCurrent()) return;
      setToast({ type: TOAST_TYPE.SUCCESS, title: "Link Copied!", message: "View link copied to clipboard." });
    } catch {
      if (!isCurrent()) return;
      setToast({ type: TOAST_TYPE.ERROR, title: "Could not copy to clipboard", message: "Please try again." });
    }
  };
}
