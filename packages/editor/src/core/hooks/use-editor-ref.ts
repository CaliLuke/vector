/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMergeRefs } from "@floating-ui/react";
import { useState, type RefCallback, type ForwardedRef } from "react";
import type { EditorRefApi } from "@/types";

/** Publish the mounted editor API to toolbars and to the caller's ref together. */
export function useEditorRef(forwardedRef: ForwardedRef<EditorRefApi>): {
  editorRef: EditorRefApi | null;
  ref: RefCallback<EditorRefApi> | null;
} {
  const [editorRef, setEditorRef] = useState<EditorRefApi | null>(null);
  const ref = useMergeRefs([forwardedRef, setEditorRef]);
  return { editorRef, ref };
}
