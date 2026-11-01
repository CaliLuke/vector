/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { HocuspocusProvider } from "@hocuspocus/provider";
import type { Editor } from "@tiptap/react";
import type { ReactNode } from "react";
import { useCallback, useEffect, useRef } from "react";
// plane utils
import { cn } from "@plane/utils/common";
// constants
import { DEFAULT_DISPLAY_CONFIG } from "@/constants/config";
import { CORE_EXTENSIONS } from "@/constants/extension";
// components
import type { TCollabValue } from "@/contexts";
import { LinkContainer } from "@/plane-editor/components/link-container";
// plugins
import { nodeHighlightPluginKey } from "@/plugins/highlight";
// types
import type { TDisplayConfig } from "@/types";

type Props = {
  children: ReactNode;
  displayConfig: TDisplayConfig;
  editor: Editor;
  editorContainerClassName: string;
  id: string;
  isTouchDevice: boolean;
  provider?: HocuspocusProvider | undefined;
  state?: TCollabValue["state"];
};

const handleContainerMouseLeave = () => {
  const dragHandleElement = document.querySelector("#editor-side-menu");
  if (!dragHandleElement?.classList.contains("side-menu-hidden")) {
    dragHandleElement?.classList.add("side-menu-hidden");
  }
};

export function EditorContainer(props: Props) {
  const { children, displayConfig, editor, editorContainerClassName, id, isTouchDevice, provider, state } = props;
  const supportsLinkPreview = !editor.isDestroyed && typeof editor.commands.setLinkPreviewOpen === "function";
  // refs
  const containerRef = useRef<HTMLDivElement>(null);
  const hasScrolledOnce = useRef(false);
  const scrollToNode = useCallback(
    (nodeId: string) => {
      if (!editor) return false;

      const doc = editor.state.doc;
      let pos: number | null = null;

      doc.descendants((node, position) => {
        if (node.attrs && node.attrs.id === nodeId) {
          pos = position;
          return false;
        }

        return undefined;
      });

      if (pos === null) {
        return false;
      }

      const nodePosition = pos;
      const tr = editor.state.tr.setMeta(nodeHighlightPluginKey, { nodeId });
      editor.view.dispatch(tr);

      requestAnimationFrame(() => {
        const domNode = editor.view.nodeDOM(nodePosition);
        if (domNode instanceof HTMLElement) {
          domNode.scrollIntoView({ behavior: "instant", block: "center" });
        }
      });

      editor.once("focus", () => {
        const clearTr = editor.state.tr.setMeta(nodeHighlightPluginKey, { nodeId: null });
        editor.view.dispatch(clearTr);
      });

      hasScrolledOnce.current = true;
      return true;
    },

    [editor]
  );

  useEffect(() => {
    const nodeId = window.location.href.split("#")[1];

    const handleSynced = () => scrollToNode(nodeId);

    if (nodeId && !hasScrolledOnce.current) {
      if (provider && state) {
        const { hasCachedContent } = state;
        // If the provider is synced or the cached content is available and the server is disconnected, scroll to the node
        if (hasCachedContent) {
          const hasScrolled = handleSynced();
          if (!hasScrolled) {
            provider.on("synced", handleSynced);
          }
        } else if (provider.isSynced) {
          handleSynced();
        } else {
          provider.on("synced", handleSynced);
        }
      } else {
        handleSynced();
      }
      return () => {
        if (provider) {
          provider.off("synced", handleSynced);
        }
      };
    }

    return undefined;
  }, [scrollToNode, provider, state]);

  const continueWriting = () => {
    if (!editor) return;
    if (!editor.isEditable) return;
    try {
      const { doc } = editor.state;
      const lastNode = doc.lastChild;
      if (
        lastNode &&
        lastNode.type.name !== CORE_EXTENSIONS.PARAGRAPH &&
        lastNode.type.name !== CORE_EXTENSIONS.DOCUMENT
      ) {
        const isStructuredBlock = [
          CORE_EXTENSIONS.ORDERED_LIST,
          CORE_EXTENSIONS.BULLET_LIST,
          CORE_EXTENSIONS.TASK_ITEM,
          CORE_EXTENSIONS.TABLE,
          CORE_EXTENSIONS.BLOCKQUOTE,
          CORE_EXTENSIONS.CODE_BLOCK,
        ].some((type) => type === lastNode.type.name);
        if (lastNode.content.size > 0 || !lastNode.isTextblock || isStructuredBlock) {
          editor.commands.insertContentAt(doc.content.size, { type: "paragraph" });
        }
      }
      editor.commands.focus("end", { scrollIntoView: false });
    } catch (error) {
      console.error("An error occurred while handling container click to insert new empty node at bottom:", error);
    }
  };

  return (
    <>
      <div
        ref={containerRef}
        id={`editor-container-${id}`}
        onMouseLeave={handleContainerMouseLeave}
        className={cn(
          `editor-container relative isolate cursor-text line-spacing-${displayConfig.lineSpacing ?? DEFAULT_DISPLAY_CONFIG.lineSpacing}`,
          {
            "active-editor": editor?.isFocused && editor?.isEditable,
          },
          displayConfig.fontSize ?? DEFAULT_DISPLAY_CONFIG.fontSize,
          displayConfig.fontStyle ?? DEFAULT_DISPLAY_CONFIG.fontStyle,
          editorContainerClassName
        )}
      >
        {editor.isEditable && (
          <button
            type="button"
            aria-label="Continue writing at end of document"
            onClick={continueWriting}
            className="absolute inset-0 -z-10 cursor-text rounded-[inherit] focus-visible:outline-2 focus-visible:outline-accent-strong"
          />
        )}
        {children}
        {!isTouchDevice && supportsLinkPreview && <LinkContainer editor={editor} containerRef={containerRef} />}
      </div>
    </>
  );
}
