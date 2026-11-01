/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor } from "@tiptap/core";

export type CommandListInstance = {
  onKeyDown: (props: { event: KeyboardEvent }) => boolean;
};

type TArgs = {
  event: KeyboardEvent;
  sections: {
    items: unknown[];
  }[];
  selectedIndex: {
    section: number;
    item: number;
  };
};

export const DROPDOWN_NAVIGATION_KEYS = ["ArrowUp", "ArrowDown", "Enter"];

export const getNextValidIndex = (
  args: TArgs
):
  | {
      section: number;
      item: number;
    }
  | undefined => {
  const { event, sections, selectedIndex } = args;
  const positions = sections.flatMap((section, sectionIndex) =>
    section.items.map((_, itemIndex) => ({ section: sectionIndex, item: itemIndex }))
  );
  if (!positions.length) return undefined;
  const current = positions.findIndex(
    ({ section, item }) => section === selectedIndex.section && item === selectedIndex.item
  );
  const direction = event.key === "ArrowUp" ? -1 : 1;
  if (current === -1) return direction === -1 ? positions.at(-1) : positions[0];
  return positions[(current + direction + positions.length) % positions.length];
};

export function createSuggestionKeyHandlers({
  getComponent,
  close,
}: {
  getComponent: () => { element: HTMLElement; ref: CommandListInstance | null } | null;
  close: (editor?: Editor) => void;
}) {
  return {
    onKeyDown: ({ event }: { event: KeyboardEvent }) => {
      if ([...DROPDOWN_NAVIGATION_KEYS, "Escape"].includes(event.key)) {
        event.preventDefault();
        event.stopPropagation();
      }
      if (event.key === "Escape") {
        close();
        return true;
      }
      return getComponent()?.ref?.onKeyDown({ event }) ?? false;
    },
    onExit: ({ editor }: { editor: Editor }) => {
      getComponent()?.element.remove();
      close(editor);
    },
  };
}
