/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observeSelectionDrag } from "@/helpers/selection-drag";
import { isNodeSelection } from "@tiptap/core";
import type { Editor } from "@tiptap/core";
import { useEditorState } from "@tiptap/react";
import { BubbleMenu } from "@tiptap/react/menus";
import type { BubbleMenuProps } from "@tiptap/react/menus";
import { useEffect, useState, useRef } from "react";
// plane utils
import { cn } from "@plane/utils/common";
// components
import type { EditorMenuItem } from "@/components/menus";
import {
  createBackgroundColorItem,
  createBoldItem,
  BubbleMenuColorSelector,
  BubbleMenuNodeSelector,
  createCodeItem,
  createItalicItem,
  createStrikeThroughItem,
  createTextAlignItem,
  createTextColorItem,
  createUnderLineItem,
} from "@/components/menus";
// constants
import { COLORS_LIST } from "@/constants/common";
import { CORE_EXTENSIONS } from "@/constants/extension";
// extensions
import { isCellSelection } from "@/extensions/table/table/utilities/helpers";
// types
import type { IEditorPropsExtended, TEditorCommands, TExtensions } from "@/types";
// local imports
import { TextAlignmentSelector } from "./alignment-selector";
import { BubbleMenuLinkSelector } from "./link-selector";

type EditorBubbleMenuProps = Omit<BubbleMenuProps, "children">;

export type EditorStateType = {
  code: boolean;
  bold: boolean;
  italic: boolean;
  underline: boolean;
  strikethrough: boolean;
  left: boolean;
  right: boolean;
  center: boolean;
  color:
    | {
        key: string;
        label: string;
        textColor: string;
        backgroundColor: string;
      }
    | undefined;
  backgroundColor:
    | {
        key: string;
        label: string;
        textColor: string;
        backgroundColor: string;
      }
    | undefined;
};

type Props = {
  disabledExtensions: TExtensions[];
  editor: Editor;
  extendedEditorProps: IEditorPropsExtended;
  flaggedExtensions: TExtensions[];
};

export function EditorBubbleMenu(props: Props) {
  const { editor } = props;
  // states
  const [isSelecting, setIsSelecting] = useState(false);
  // refs
  const menuRef = useRef<HTMLDivElement>(null);

  const formattingItems = {
    code: createCodeItem(editor),
    bold: createBoldItem(editor),
    italic: createItalicItem(editor),
    underline: createUnderLineItem(editor),
    strikethrough: createStrikeThroughItem(editor),
    "text-align": createTextAlignItem(editor),
  } satisfies {
    [K in TEditorCommands]?: EditorMenuItem<K>;
  };

  const editorState: EditorStateType = useEditorState({
    editor,
    selector: ({ editor: currentEditor }) => ({
      code: formattingItems.code.isActive(),
      bold: formattingItems.bold.isActive(),
      italic: formattingItems.italic.isActive(),
      underline: formattingItems.underline.isActive(),
      strikethrough: formattingItems.strikethrough.isActive(),
      left: formattingItems["text-align"].isActive({ alignment: "left" }),
      right: formattingItems["text-align"].isActive({ alignment: "right" }),
      center: formattingItems["text-align"].isActive({ alignment: "center" }),
      color: COLORS_LIST.find((c) => createTextColorItem(currentEditor).isActive({ color: c.key })),
      backgroundColor: COLORS_LIST.find((c) => createBackgroundColorItem(currentEditor).isActive({ color: c.key })),
    }),
  });

  const basicFormattingOptions = editorState.code
    ? [formattingItems.code]
    : [formattingItems.bold, formattingItems.italic, formattingItems.underline, formattingItems.strikethrough];

  const bubbleMenuProps: EditorBubbleMenuProps = {
    editor,
    style: { zIndex: 9 },
    shouldShow: ({ state, editor: currentEditor }) => {
      const { selection } = state;
      const { empty } = selection;

      if (
        empty ||
        !currentEditor.isEditable ||
        currentEditor.isActive(CORE_EXTENSIONS.IMAGE) ||
        currentEditor.isActive(CORE_EXTENSIONS.CUSTOM_IMAGE) ||
        isNodeSelection(selection) ||
        isCellSelection(selection) ||
        isSelecting
      ) {
        return false;
      }
      return true;
    },
    options: {
      onShow: () => {
        editor.commands.setLinkBubbleMenuOpen?.(true);
        editor.commands.addActiveDropbarExtension("bubble-menu");
      },
      onHide: () => {
        editor.commands.setLinkBubbleMenuOpen?.(false);
        setTimeout(() => {
          editor.commands.removeActiveDropbarExtension("bubble-menu");
        }, 0);
      },
    },
  };

  useEffect(() => observeSelectionDrag(editor, () => menuRef.current, setIsSelecting), [editor]);

  return (
    <BubbleMenu {...bubbleMenuProps}>
      {!isSelecting && (
        <div
          ref={menuRef}
          className="horizontal-scrollbar flex scrollbar-xs divide-x divide-subtle-1 overflow-x-scroll rounded-lg border border-subtle bg-surface-1 py-2 shadow-raised-200"
        >
          <div className="px-2">
            <BubbleMenuNodeSelector editor={editor} />
          </div>
          {!editorState.code && (
            <div className="px-2">
              <BubbleMenuLinkSelector editor={editor} />
            </div>
          )}
          {!editorState.code && (
            <div className="px-2">
              <BubbleMenuColorSelector editor={editor} editorState={editorState} />
            </div>
          )}
          <div className="flex gap-0.5 px-2">
            {basicFormattingOptions.map((item) => (
              <button
                key={item.key}
                aria-label={item.name}
                aria-pressed={editorState[item.key]}
                type="button"
                onClick={(e) => {
                  item.command();
                  e.stopPropagation();
                }}
                className={cn(
                  "grid size-7 place-items-center rounded-sm text-tertiary transition-colors hover:bg-layer-1 active:bg-layer-1",
                  {
                    "bg-layer-1 text-primary": editorState[item.key],
                  }
                )}
              >
                <item.icon className="size-4" />
              </button>
            ))}
          </div>
          <TextAlignmentSelector editor={editor} editorState={editorState} />
        </div>
      )}
    </BubbleMenu>
  );
}
