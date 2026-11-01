/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useCallback } from "react";
import { EmojiRoot } from "../emoji-icon-picker/emoji/emoji";
import { emojiToString } from "../emoji-icon-picker/helper";
import { Popover } from "../popover";
import { cn } from "@plane/utils/common";
import type { TPlacement, TSide, TAlign } from "../utils/placement";
import { EmojiReactionButton, EmojiReactionGroup } from "./emoji-reaction";
import type { EmojiReactionButtonProps, EmojiReactionGroupProps } from "./emoji-reaction";

export interface EmojiReactionPickerProps {
  isOpen: boolean;
  handleToggle: (value: boolean) => void;
  buttonClassName?: string;
  closeOnSelect?: boolean;
  disabled?: boolean;
  dropdownClassName?: string;
  label: React.ReactNode;
  onChange: (emoji: string) => void;
  placement?: TPlacement;
  searchDisabled?: boolean;
  searchPlaceholder?: string;
  side?: TSide;
  align?: TAlign;
}

export function EmojiReactionPicker(props: EmojiReactionPickerProps) {
  const {
    isOpen,
    handleToggle,
    buttonClassName,
    closeOnSelect = true,
    disabled = false,
    dropdownClassName,
    label,
    onChange,
    placement,
    searchDisabled = false,
    searchPlaceholder = "Search",
    side = "bottom",
    align = "start",
  } = props;

  const handleEmojiChange = useCallback(
    (value: string) => {
      const emoji = emojiToString(value);
      onChange(emoji);
      if (closeOnSelect) handleToggle(false);
    },
    [onChange, closeOnSelect, handleToggle]
  );

  const triggerClassName = cn("outline-none", buttonClassName);
  const isReactionGroup = React.isValidElement<EmojiReactionGroupProps>(label) && label.type === EmojiReactionGroup;
  const isButtonLabel =
    React.isValidElement<Record<string, unknown>>(label) &&
    (label.type === "button" || label.type === EmojiReactionButton);

  const trigger = isReactionGroup ? (
    React.cloneElement(label, {
      renderAddButton: (button: React.ReactElement<EmojiReactionButtonProps>) => (
        <Popover.Button
          className={triggerClassName}
          disabled={disabled}
          render={(triggerProps) => React.cloneElement(button, { ...triggerProps, onAddReaction: undefined })}
        />
      ),
    })
  ) : isButtonLabel ? (
    <Popover.Button className={triggerClassName} disabled={disabled} render={label} />
  ) : (
    <Popover.Button className={triggerClassName} disabled={disabled}>
      {label}
    </Popover.Button>
  );

  return (
    <Popover open={isOpen} onOpenChange={handleToggle}>
      {trigger}
      <Popover.Panel
        positionerClassName="z-50"
        className={cn("w-80 overflow-hidden rounded-md border-[0.5px] border-strong bg-surface-1", dropdownClassName)}
        placement={placement}
        side={side}
        align={align}
        sideOffset={8}
        data-prevent-outside-click="true"
      >
        <div className="h-80 overflow-hidden overflow-y-auto">
          <EmojiRoot
            onChange={handleEmojiChange}
            searchPlaceholder={searchPlaceholder}
            searchDisabled={searchDisabled}
          />
        </div>
      </Popover.Panel>
    </Popover>
  );
}
