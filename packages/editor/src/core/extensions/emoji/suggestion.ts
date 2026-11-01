/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { EmojiOptions } from "@tiptap/extension-emoji";
import { ReactRenderer } from "@tiptap/react";
import type { Editor } from "@tiptap/react";
// constants
import { CORE_EXTENSIONS } from "@/constants/extension";
// helpers
import { updateFloatingUIFloaterPosition } from "@/helpers/floating-ui";
import type { CommandListInstance } from "@/helpers/tippy";
import { createSuggestionKeyHandlers } from "@/helpers/tippy";
// local imports
import { EmojisListDropdown } from "./components/emojis-list";
import type { EmojisListDropdownProps, EmojiItem } from "./components/emojis-list";
import type { ExtendedEmojiStorage } from "./emoji";

const DEFAULT_EMOJIS = ["+1", "-1", "smile", "orange_heart", "eyes"];

export const emojiSuggestion: EmojiOptions["suggestion"] = {
  items: ({ editor, query }: { editor: Editor; query: string }): EmojiItem[] => {
    const { emojis } = editor.storage.emoji;

    if (query.trim() === "") {
      const defaultEmojis = DEFAULT_EMOJIS.map((name) =>
        emojis.find((emoji) => emoji.shortcodes.includes(name) || emoji.name === name)
      )
        .filter((emoji) => emoji !== undefined)
        .slice(0, 5);
      return defaultEmojis;
    }
    return emojis
      .filter(({ shortcodes, tags }) => {
        const lowerQuery = query.toLowerCase();
        return (
          shortcodes.find((shortcode: string) => shortcode.startsWith(lowerQuery)) ||
          tags.find((tag: string) => tag.startsWith(lowerQuery))
        );
      })
      .slice(0, 5);
  },

  allowSpaces: false,

  render: () => {
    let component: ReactRenderer<CommandListInstance, EmojisListDropdownProps> | null = null;
    let cleanup: (() => void) | undefined;
    let editorRef: Editor | null = null;

    const handleClose = (editor?: Editor) => {
      component?.destroy();
      component = null;
      const activeEditor = editor ?? editorRef;
      activeEditor?.commands.removeActiveDropbarExtension(CORE_EXTENSIONS.EMOJI);
      const emojiStorage: Partial<ExtendedEmojiStorage> | undefined = activeEditor?.storage.emoji;
      if (emojiStorage) emojiStorage.forceOpen = false;
      cleanup?.();
    };

    return {
      onStart: (props) => {
        editorRef = props.editor;
        const emojiStorage: Partial<ExtendedEmojiStorage> = props.editor.storage.emoji;
        const forceOpen = emojiStorage.forceOpen || false;
        component = new ReactRenderer<CommandListInstance, EmojisListDropdownProps>(EmojisListDropdown, {
          props: {
            ...props,
            onClose: () => handleClose(props.editor),
            forceOpen,
          } satisfies EmojisListDropdownProps,
          editor: props.editor,
          className: "fixed z-[100]",
        });
        if (!props.clientRect) return;
        props.editor.commands.addActiveDropbarExtension(CORE_EXTENSIONS.EMOJI);
        const element = component.element;
        cleanup = updateFloatingUIFloaterPosition(props.editor, element).cleanup;
      },

      onUpdate: (props) => {
        if (!component || !component.element) return;
        const emojiStorage: Partial<ExtendedEmojiStorage> = props.editor.storage.emoji;
        const forceOpen = emojiStorage.forceOpen || false;
        component.updateProps({ ...props, forceOpen });
        if (!props.clientRect) return;
        cleanup?.();
        cleanup = updateFloatingUIFloaterPosition(props.editor, component.element).cleanup;
      },
      ...createSuggestionKeyHandlers({ getComponent: () => component, close: handleClose }),
    };
  },
};
