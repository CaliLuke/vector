/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { NodeViewProps } from "@tiptap/react";
import { NodeViewContent, NodeViewWrapper } from "@tiptap/react";
import { useState } from "react";
// constants
import { COLORS_LIST } from "@/constants/common";
// local components
import { CalloutBlockColorSelector } from "./color-selector";
import { CalloutBlockLogoSelector } from "./logo-selector";
// types
import type { TCalloutBlockAttributes } from "./types";
import { ECalloutAttributeNames } from "./types";
// utils
import { getCalloutAttributes, updateStoredBackgroundColor } from "./utils";

export type CustomCalloutNodeViewProps = NodeViewProps & {
  updateAttributes: (attrs: Partial<TCalloutBlockAttributes>) => void;
};

export function CustomCalloutBlock(props: CustomCalloutNodeViewProps) {
  const { editor, node, updateAttributes } = props;
  const attrs = getCalloutAttributes(node.attrs);
  // states
  const [isEmojiPickerOpen, setIsEmojiPickerOpen] = useState(false);
  const [isColorPickerOpen, setIsColorPickerOpen] = useState(false);
  // derived values
  const activeBackgroundColor = COLORS_LIST.find((c) => attrs["data-background"] === c.key)?.backgroundColor;

  return (
    <NodeViewWrapper
      key={attrs[ECalloutAttributeNames.ID]}
      className="editor-callout-component group/callout-node relative my-2 flex items-start gap-4 rounded-lg bg-layer-3 p-4 break-words text-primary transition-colors duration-500"
      style={{
        backgroundColor: activeBackgroundColor,
      }}
    >
      <CalloutBlockLogoSelector
        key={attrs[ECalloutAttributeNames.ID]}
        blockAttributes={attrs}
        disabled={!editor.isEditable}
        isOpen={isEmojiPickerOpen}
        handleOpen={(val) => setIsEmojiPickerOpen(val)}
        updateAttributes={updateAttributes}
      />
      <CalloutBlockColorSelector
        disabled={!editor.isEditable}
        isOpen={isColorPickerOpen}
        toggleDropdown={() => setIsColorPickerOpen((prev) => !prev)}
        onSelect={(val) => {
          updateAttributes({
            [ECalloutAttributeNames.BACKGROUND]: val,
          });
          updateStoredBackgroundColor(val);
        }}
      />
      <NodeViewContent as="div" className="w-full break-words" />
    </NodeViewWrapper>
  );
}
