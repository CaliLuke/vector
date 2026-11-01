/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor } from "@tiptap/react";

import { CheckIcon, ChevronDownIcon } from "@plane/propel/icons";
// plane utils
import { cn } from "@plane/utils/common";
// components
import {
  createBulletListItem,
  createHeadingOneItem,
  createHeadingThreeItem,
  createHeadingTwoItem,
  createNumberedListItem,
  createQuoteItem,
  createCodeItem,
  createTodoListItem,
  createTextItem,
  createHeadingFourItem,
  createHeadingFiveItem,
  createHeadingSixItem,
} from "@/components/menus";
// types
// local imports
import { FloatingMenuRoot } from "../floating-menu/root";
import { useFloatingMenu } from "../floating-menu/use-floating-menu";

type Props = {
  editor: Editor;
};

export function BubbleMenuNodeSelector(props: Props) {
  const { editor } = props;
  // floating ui
  const { options, getReferenceProps, getFloatingProps } = useFloatingMenu({});
  const { context } = options;
  const items = [
    createTextItem(editor),
    createHeadingOneItem(editor),
    createHeadingTwoItem(editor),
    createHeadingThreeItem(editor),
    createHeadingFourItem(editor),
    createHeadingFiveItem(editor),
    createHeadingSixItem(editor),
    createBulletListItem(editor),
    createNumberedListItem(editor),
    createTodoListItem(editor),
    createQuoteItem(editor),
    createCodeItem(editor),
  ];

  const activeItem = items.findLast((item) => item.isActive()) ?? {
    name: "Multiple",
  };

  return (
    <FloatingMenuRoot
      classNames={{
        buttonContainer: "h-full",
        button: cn(
          "flex h-full items-center gap-1 rounded-sm px-3 text-13 font-medium whitespace-nowrap text-tertiary transition-colors hover:bg-layer-1 active:bg-layer-1",
          {
            "bg-layer-1": context.open,
          }
        ),
      }}
      menuButton={
        <>
          <span>{activeItem?.name}</span>
          <ChevronDownIcon className="size-3 shrink-0" />
        </>
      }
      options={options}
      getFloatingProps={getFloatingProps}
      getReferenceProps={getReferenceProps}
    >
      <section className="mt-1 flex max-h-[90vh] w-48 flex-col overflow-y-scroll rounded-md border-[0.5px] border-strong bg-surface-1 px-2 py-2.5 shadow-raised-200">
        {items.map((item) => (
          <button
            key={item.name}
            type="button"
            onClick={(e) => {
              item.command();
              context.onOpenChange(false);
              e.stopPropagation();
            }}
            className={cn(
              "flex items-center justify-between rounded-sm px-1 py-1.5 text-13 text-secondary hover:bg-layer-1",
              {
                "bg-layer-1": activeItem.name === item.name,
              }
            )}
          >
            <div className="flex items-center space-x-2">
              <item.icon className="size-3 flex-shrink-0" />
              <span>{item.name}</span>
            </div>
            {activeItem.name === item.name && <CheckIcon className="size-3 flex-shrink-0 text-tertiary" />}
          </button>
        ))}
      </section>
    </FloatingMenuRoot>
  );
}
