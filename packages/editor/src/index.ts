/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// editors
export {
  CollaborativeDocumentEditorWithRef,
  DocumentEditorWithRef,
  LiteTextEditorWithRef,
  RichTextEditorWithRef,
} from "@/components/editors";

// constants
export * from "@/constants/common";

// helpers
export * from "@/helpers/common";
export { processAssetDuplication } from "@/helpers/paste-asset";
export * from "@/helpers/yjs-utils";

export { CORE_EXTENSIONS } from "@/constants/extension";
export { ADDITIONAL_EXTENSIONS } from "@/plane-editor/constants/extensions";

// types
export * from "@/types";

// additional exports
export { useEditorRef } from "./core/hooks/use-editor-ref";
export { useEditorToolbar } from "./core/hooks/use-editor-toolbar";
export { executeToolbarCommand } from "./core/helpers/toolbar-command";
export { EditorToolbarButtons } from "./core/components/menus/toolbar-buttons";
export { TrailingNode } from "./core/extensions/trailing-node";
