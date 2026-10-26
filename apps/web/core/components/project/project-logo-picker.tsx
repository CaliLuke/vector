/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useState } from "react";
import { EmojiPicker, EmojiIconPickerTypes, Logo } from "@plane/propel/emoji-icon-picker";
import type { TLogoProps } from "@plane/types";
export function ProjectLogoPicker({
  value,
  onChange,
  disabled,
}: {
  value: TLogoProps;
  onChange: (value: TLogoProps) => void;
  disabled?: boolean;
}) {
  const [open, setOpen] = useState(false);
  return (
    <EmojiPicker
      iconType="material"
      isOpen={open}
      handleToggle={setOpen}
      disabled={disabled}
      label={
        <span className="grid size-11 place-items-center rounded-md border border-subtle bg-layer-2">
          <Logo logo={value} size={24} />
        </span>
      }
      defaultIconColor={value.icon?.color}
      defaultOpen={value.in_use === "emoji" ? EmojiIconPickerTypes.EMOJI : EmojiIconPickerTypes.ICON}
      onChange={(selection) => {
        onChange(
          selection.type === "emoji"
            ? { in_use: "emoji", emoji: { value: selection.value } }
            : { in_use: "icon", icon: selection.value }
        );
        setOpen(false);
      }}
    />
  );
}
