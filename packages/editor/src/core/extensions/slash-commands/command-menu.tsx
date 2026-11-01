/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { FloatingOverlay } from "@floating-ui/react";
import type { SuggestionProps } from "@tiptap/suggestion";
import { forwardRef, useCallback, useEffect, useImperativeHandle, useLayoutEffect, useRef, useState } from "react";
// plane imports
import { useOutsideClickDetector } from "@plane/hooks";
// helpers
import { DROPDOWN_NAVIGATION_KEYS, getNextValidIndex } from "@/helpers/tippy";
// types
import type { ISlashCommandItem } from "@/types";
// components
import type { TSlashCommandSection } from "./command-items-list";
import { CommandMenuItem } from "./command-menu-item";

export type SlashCommandsMenuProps = Pick<
  SuggestionProps<TSlashCommandSection, ISlashCommandItem>,
  "items" | "command" | "query"
> & {
  onClose: () => void;
};

export const SlashCommandsMenu = forwardRef(function SlashCommandsMenu(props: SlashCommandsMenuProps, ref) {
  const { items: sections, command, query, onClose } = props;
  // states
  const [selectedIndex, setSelectedIndex] = useState({
    section: 0,
    item: 0,
  });
  const [previousSections, setPreviousSections] = useState(sections);
  if (previousSections !== sections) {
    setPreviousSections(sections);
    setSelectedIndex({ section: 0, item: 0 });
  }
  // refs
  const commandListContainer = useRef<HTMLDivElement>(null);

  const selectItem = useCallback(
    (sectionIndex: number, itemIndex: number) => {
      const item = sections[sectionIndex]?.items?.[itemIndex];
      if (item) command(item);
    },
    [command, sections]
  );
  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (!DROPDOWN_NAVIGATION_KEYS.includes(event.key)) return false;
      event.preventDefault();
      if (event.key === "Enter") {
        selectItem(selectedIndex.section, selectedIndex.item);
      } else {
        const next = getNextValidIndex({ event, sections, selectedIndex });
        if (next) setSelectedIndex(next);
      }
      return true;
    },
    [sections, selectedIndex, selectItem]
  );

  // The editor's suggestion handler consumes its own events. Document fallback
  // handles keyboard navigation when focus is in the dropdown itself.
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (!event.defaultPrevented) handleKeyDown(event);
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [handleKeyDown]);

  // scroll to the dropdown item when navigating via keyboard
  useLayoutEffect(() => {
    const container = commandListContainer?.current;
    if (!container) return;

    const item = container.querySelector(`#item-${selectedIndex.section}-${selectedIndex.item}`);

    // use scroll into view to bring the item in view if it is not in view
    item?.scrollIntoView({ block: "nearest" });
  }, [selectedIndex]);

  useImperativeHandle(
    ref,
    () => ({
      onKeyDown: ({ event }: { event: KeyboardEvent }) => handleKeyDown(event),
    }),
    [handleKeyDown]
  );

  useOutsideClickDetector(commandListContainer, onClose);

  const areSearchResultsEmpty = sections.map((s) => s.items?.length).reduce((acc, curr) => acc + curr, 0) === 0;

  if (areSearchResultsEmpty) return null;

  return (
    <>
      {/* Backdrop */}
      <FloatingOverlay
        style={{
          zIndex: 99,
        }}
        lockScroll
      />
      <div
        role="presentation"
        id="slash-command"
        ref={commandListContainer}
        className="relative max-h-80 min-w-[12rem] space-y-2 overflow-y-auto rounded-md border-[0.5px] border-strong bg-surface-1 px-2 py-2.5 shadow-raised-200"
        style={{
          zIndex: 100,
        }}
        onClick={(e) => {
          e.stopPropagation();
        }}
        onMouseDown={(e) => {
          e.stopPropagation();
        }}
      >
        {sections.map((section, sectionIndex) => (
          <div key={section.key} className="space-y-2">
            {section.title && <h6 className="text-11 font-semibold text-tertiary">{section.title}</h6>}
            <div>
              {section.items?.map((item, itemIndex) => (
                <CommandMenuItem
                  key={item.key}
                  isSelected={sectionIndex === selectedIndex.section && itemIndex === selectedIndex.item}
                  item={item}
                  itemIndex={itemIndex}
                  onClick={(e) => {
                    e.stopPropagation();
                    selectItem(sectionIndex, itemIndex);
                  }}
                  onMouseEnter={() =>
                    setSelectedIndex({
                      section: sectionIndex,
                      item: itemIndex,
                    })
                  }
                  sectionIndex={sectionIndex}
                  query={query}
                />
              ))}
            </div>
          </div>
        ))}
      </div>
    </>
  );
});

SlashCommandsMenu.displayName = "SlashCommandsMenu";
