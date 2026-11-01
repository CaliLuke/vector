/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useState } from "react";
import { Palette } from "lucide-react";
// editor
import type { EditorRefApi } from "@plane/editor";
// ui
import { useOutsideClickDetector } from "@plane/hooks";
import { TrashIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import type { TSticky } from "@plane/types";
// constants
import type { ToolbarMenuItem } from "@plane/editor";
import { TOOLBAR_ITEMS, EditorToolbarButtons, useEditorToolbar } from "@plane/editor";
// helpers
import { ColorPalette } from "./color-palette";

type Props = {
  executeCommand: (item: ToolbarMenuItem) => void;
  editorRef: EditorRefApi | null;
  handleColorChange: (data: Partial<TSticky>) => Promise<void>;
  handleDelete: () => void;
};

const toolbarItems = TOOLBAR_ITEMS.sticky;

export function StickyEditorToolbar(props: Props) {
  const { executeCommand, editorRef, handleColorChange, handleDelete } = props;

  const { activeStates } = useEditorToolbar(editorRef, toolbarItems);
  const [showColorPalette, setShowColorPalette] = useState(false);
  const colorPaletteRef = React.useRef<HTMLDivElement>(null);

  useOutsideClickDetector(colorPaletteRef, () => setShowColorPalette(false));

  return (
    <div className="flex h-full w-full justify-between">
      <div className="my-auto flex gap-4" ref={colorPaletteRef}>
        {/* color palette */}
        {showColorPalette && <ColorPalette handleUpdate={handleColorChange} />}
        <Tooltip
          tooltipContent={
            <p className="flex flex-col gap-1 text-center text-11">
              <span className="font-medium">Background color</span>
            </p>
          }
        >
          <button
            type="button"
            aria-label="Background color"
            aria-expanded={showColorPalette}
            onClick={() => setShowColorPalette(!showColorPalette)}
            className="flex text-primary/50"
          >
            <Palette className="my-auto size-4" />
          </button>
        </Tooltip>

        <div className="my-auto flex w-fit items-stretch justify-between gap-4 rounded-sm p-1">
          <div className="my-auto flex items-stretch gap-4">
            <EditorToolbarButtons
              groups={toolbarItems}
              activeStates={activeStates}
              executeCommand={executeCommand}
              variant="sticky"
            />
          </div>
        </div>
      </div>
      {/* delete action */}
      <Tooltip
        tooltipContent={
          <p className="flex flex-col gap-1 text-center text-11">
            <span className="font-medium">Delete</span>
          </p>
        }
      >
        <button type="button" aria-label="Delete sticky" onClick={handleDelete} className="my-auto text-primary/50">
          <TrashIcon className="size-4" />
        </button>
      </Tooltip>
    </div>
  );
}
