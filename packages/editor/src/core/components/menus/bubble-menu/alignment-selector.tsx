/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor } from "@tiptap/core";
import type { LucideIcon } from "lucide-react";
import { AlignCenter, AlignLeft, AlignRight } from "lucide-react";
// plane utils
import { cn } from "@plane/utils/common";
// components
import { createTextAlignItem } from "@/components/menus";
// types
import type { TEditorCommands } from "@/types";
import type { EditorStateType } from "./root";

type Props = {
  editor: Editor;
  editorState: EditorStateType;
};

export function TextAlignmentSelector(props: Props) {
  const { editor, editorState } = props;
  const menuItem = createTextAlignItem(editor);

  const textAlignmentOptions: {
    itemKey: TEditorCommands;
    renderKey: string;
    label: string;
    icon: LucideIcon;
    command: () => void;
    isActive: () => boolean;
  }[] = [
    {
      itemKey: "text-align",
      renderKey: "text-align-left",
      label: "Align left",
      icon: AlignLeft,
      command: () =>
        menuItem.command({
          alignment: "left",
        }),
      isActive: () => editorState.left,
    },
    {
      itemKey: "text-align",
      renderKey: "text-align-center",
      label: "Align center",
      icon: AlignCenter,
      command: () =>
        menuItem.command({
          alignment: "center",
        }),
      isActive: () => editorState.center,
    },
    {
      itemKey: "text-align",
      renderKey: "text-align-right",
      label: "Align right",
      icon: AlignRight,
      command: () =>
        menuItem.command({
          alignment: "right",
        }),
      isActive: () => editorState.right,
    },
  ];
  if (editorState.code) return null;

  return (
    <div className="flex gap-0.5 px-2">
      {textAlignmentOptions.map((item) => (
        <button
          key={item.renderKey}
          aria-label={item.label}
          aria-pressed={item.isActive()}
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            item.command();
          }}
          className={cn(
            "grid size-7 place-items-center rounded-sm text-tertiary transition-colors hover:bg-layer-1 active:bg-layer-1",
            {
              "bg-layer-1 text-primary": item.isActive(),
            }
          )}
        >
          <item.icon className="size-4" />
        </button>
      ))}
    </div>
  );
}
