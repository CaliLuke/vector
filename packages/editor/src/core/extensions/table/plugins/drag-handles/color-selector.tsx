/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Disclosure } from "@headlessui/react";
import type { Editor } from "@tiptap/core";
import { Palette } from "lucide-react";
// plane imports
import { ChevronRightIcon } from "@plane/propel/icons";
import { cn } from "@plane/utils/common";
// constants
import { ColorSwatches } from "@/components/menus/color-swatches";
import { CORE_EXTENSIONS } from "@/constants/extension";

type Props = {
  editor: Editor;
  onSelect: (color: string | null) => void;
};

const handleBackgroundColorChange = (editor: Editor, color: string | null) => {
  editor
    .chain()
    .focus()
    .updateAttributes(CORE_EXTENSIONS.TABLE_CELL, {
      background: color,
    })
    .updateAttributes(CORE_EXTENSIONS.TABLE_HEADER, { background: color })
    .run();
};

export function TableDragHandleDropdownColorSelector(props: Props) {
  const { editor, onSelect } = props;

  return (
    <Disclosure defaultOpen>
      <Disclosure.Button
        as="button"
        type="button"
        className="flex w-full items-center justify-between gap-2 truncate rounded-sm px-1 py-1.5 text-left text-11 text-secondary hover:bg-layer-1"
      >
        {({ open }) => (
          <>
            <span className="flex items-center gap-2">
              <Palette className="size-3 shrink-0" />
              Color
            </span>
            <ChevronRightIcon
              className={cn("size-3 shrink-0 transition-transform duration-200", {
                "rotate-90": open,
              })}
            />
          </>
        )}
      </Disclosure.Button>
      <Disclosure.Panel className="mb-1.5 space-y-2 p-1">
        <div className="space-y-1">
          <p className="text-11 font-semibold text-tertiary">Background colors</p>
          <div className="flex flex-wrap items-center gap-2">
            <ColorSwatches
              tone="background"
              resetClassName="hover:bg-layer-1-hover"
              onSelect={(color) => {
                if (editor.isDestroyed || !editor.isEditable) return;
                const background = color?.backgroundColor ?? null;
                handleBackgroundColorChange(editor, background);
                onSelect(background);
              }}
            />
          </div>
        </div>
      </Disclosure.Panel>
    </Disclosure>
  );
}
