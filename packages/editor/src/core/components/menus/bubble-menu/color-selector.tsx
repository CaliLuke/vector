/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor } from "@tiptap/react";
import { ALargeSmall } from "lucide-react";
// plane utils
import { cn } from "@plane/utils/common";
// constants
import { ColorSwatches } from "../color-swatches";
// local imports
import { FloatingMenuRoot } from "../floating-menu/root";
import { useFloatingMenu } from "../floating-menu/use-floating-menu";
import { createBackgroundColorItem, createTextColorItem } from "../menu-items";
import type { EditorStateType } from "./root";

type Props = {
  editor: Editor;
  editorState: EditorStateType;
};

export function BubbleMenuColorSelector(props: Props) {
  const { editor, editorState } = props;
  // floating ui
  const { options, getReferenceProps, getFloatingProps } = useFloatingMenu({});

  const activeTextColor = editorState.color;
  const activeBackgroundColor = editorState.backgroundColor;

  return (
    <FloatingMenuRoot
      classNames={{
        buttonContainer: "h-full",
        button:
          "flex items-center gap-1 h-full whitespace-nowrap px-3 text-13 font-medium text-tertiary hover:bg-layer-1 active:bg-layer-1 rounded-sm transition-colors",
      }}
      menuButton={
        <>
          <span>Color</span>
          <span
            className={cn("grid size-6 flex-shrink-0 place-items-center rounded-sm border-[0.5px] border-strong", {
              "bg-surface-1": !activeBackgroundColor,
            })}
            style={{
              backgroundColor: activeBackgroundColor ? activeBackgroundColor.backgroundColor : "transparent",
            }}
          >
            <ALargeSmall
              className={cn("size-3.5", {
                "text-primary": !activeTextColor,
              })}
              style={{
                color: activeTextColor ? activeTextColor.textColor : "inherit",
              }}
            />
          </span>
        </>
      }
      options={options}
      getFloatingProps={getFloatingProps}
      getReferenceProps={getReferenceProps}
    >
      <section className="mt-1 space-y-2 rounded-md border-[0.5px] border-strong bg-surface-1 p-2 shadow-raised-200">
        <div className="space-y-1.5">
          <p className="text-11 font-semibold text-tertiary">Text colors</p>
          <div className="flex items-center gap-2">
            <ColorSwatches
              tone="text"
              onSelect={(color) => createTextColorItem(editor).command({ color: color?.key })}
            />
          </div>
        </div>
        <div className="space-y-1.5">
          <p className="text-11 font-semibold text-tertiary">Background colors</p>
          <div className="flex items-center gap-2">
            <ColorSwatches
              tone="background"
              onSelect={(color) => createBackgroundColorItem(editor).command({ color: color?.key })}
            />
          </div>
        </div>
      </section>
    </FloatingMenuRoot>
  );
}
