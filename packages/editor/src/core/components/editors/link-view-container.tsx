/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { autoUpdate, flip, hide, shift, useDismiss, useFloating, useInteractions } from "@floating-ui/react";
import type { Editor } from "@tiptap/react";
import { useCallback, useEffect, useRef, useState } from "react";
import type { LinkViewProps } from "@/components/links";
import { LinkView } from "@/components/links";

type Props = {
  editor: Editor;
  containerRef: React.RefObject<HTMLDivElement | null>;
};

function getLinkAtTarget(editor: Editor, target: EventTarget | null) {
  const anchor = target instanceof Element ? target.closest("a") : null;
  if (!anchor) return undefined;
  const pos = editor.view.posAtDOM(anchor, 0);
  if (pos < 0) return undefined;
  const node = editor.state.doc.nodeAt(pos);
  const mark = node?.marks.find((candidate) => candidate.type.name === "link");
  if (!node || typeof mark?.attrs.href !== "string") return undefined;
  return { anchor, pos, node, href: mark.attrs.href };
}

export function LinkViewContainer({ editor, containerRef }: Props) {
  const [linkViewProps, setLinkViewProps] = useState<LinkViewProps>();
  const [isOpen, setIsOpen] = useState(false);
  const [reference, setReference] = useState<Element | null>(null);
  const hoverTimeoutRef = useRef<number | null>(null);
  const close = useCallback(() => {
    setIsOpen(false);
    if (!editor.isDestroyed) editor.commands.setLinkPreviewOpen(false);
  }, [editor]);
  const { refs, floatingStyles, context } = useFloating({
    open: isOpen,
    onOpenChange: (open) => {
      setIsOpen(open);
      if (!editor.isDestroyed) editor.commands.setLinkPreviewOpen(open);
    },
    elements: { reference },
    middleware: [flip({ fallbackPlacements: ["top", "bottom"] }), shift({ padding: 5 }), hide()],
    whileElementsMounted: autoUpdate,
    placement: "bottom-start",
  });
  const floating = refs.floating;
  const setFloating = useCallback((node: HTMLDivElement | null) => refs.setFloating(node), [refs]);
  const dismiss = useDismiss(context);
  const { getFloatingProps } = useInteractions([dismiss]);
  const clearHoverTimeout = useCallback(() => {
    if (hoverTimeoutRef.current !== null) {
      window.clearTimeout(hoverTimeoutRef.current);
      hoverTimeoutRef.current = null;
    }
  }, []);
  const setCloseTimeout = useCallback(() => {
    clearHoverTimeout();
    hoverTimeoutRef.current = window.setTimeout(close, 400);
  }, [clearHoverTimeout, close]);
  const handleLinkEnter = useCallback(
    (event: MouseEvent | FocusEvent) => {
      if (editor.isDestroyed || editor.storage.link.isBubbleMenuOpen) return;
      try {
        const link = getLinkAtTarget(editor, event.target);
        if (!link) return;
        clearHoverTimeout();
        setReference(link.anchor);
        setLinkViewProps({
          view: "LinkPreview",
          url: link.href,
          text: link.node.text ?? "",
          editor,
          from: link.pos,
          to: link.pos + link.node.nodeSize,
          closeLinkView: close,
        });
        setIsOpen(true);
        editor.commands.setLinkPreviewOpen(true);
      } catch (error) {
        console.error("Error handling link hover:", error);
      }
    },
    [editor, clearHoverTimeout, close]
  );
  const handleContainerLeave = useCallback(
    (event: MouseEvent | FocusEvent) => {
      const next = event.relatedTarget;
      if (next instanceof Node && (containerRef.current?.contains(next) || floating.current?.contains(next))) return;
      setCloseTimeout();
    },
    [containerRef, floating, setCloseTimeout]
  );

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return undefined;
    container.addEventListener("mouseover", handleLinkEnter);
    container.addEventListener("focusin", handleLinkEnter);
    container.addEventListener("mouseenter", clearHoverTimeout);
    container.addEventListener("mouseleave", handleContainerLeave);
    container.addEventListener("focusout", handleContainerLeave);
    return () => {
      container.removeEventListener("mouseover", handleLinkEnter);
      container.removeEventListener("focusin", handleLinkEnter);
      container.removeEventListener("mouseenter", clearHoverTimeout);
      container.removeEventListener("mouseleave", handleContainerLeave);
      container.removeEventListener("focusout", handleContainerLeave);
    };
  }, [containerRef, handleLinkEnter, clearHoverTimeout, handleContainerLeave]);
  useEffect(() => {
    const onBubbleMenuChange = ({ isOpen: bubbleOpen }: { isOpen: boolean }) => {
      if (bubbleOpen) {
        clearHoverTimeout();
        close();
      }
    };
    editor.on("linkBubbleMenuChange", onBubbleMenuChange);
    return () => {
      editor.off("linkBubbleMenuChange", onBubbleMenuChange);
    };
  }, [editor, close, clearHoverTimeout]);
  useEffect(
    () => () => {
      clearHoverTimeout();
      if (!editor.isDestroyed) editor.commands.setLinkPreviewOpen(false);
    },
    [editor, clearHoverTimeout]
  );

  return isOpen && linkViewProps && reference ? (
    <div
      ref={setFloating}
      style={{ ...floatingStyles, zIndex: 100 }}
      {...getFloatingProps()}
      onMouseEnter={clearHoverTimeout}
      onMouseLeave={setCloseTimeout}
      onFocus={clearHoverTimeout}
      onBlur={(event) => {
        if (!(event.relatedTarget instanceof Node) || !event.currentTarget.contains(event.relatedTarget))
          setCloseTimeout();
      }}
    >
      <LinkView {...linkViewProps} focusContext={context} />
    </div>
  ) : null;
}
