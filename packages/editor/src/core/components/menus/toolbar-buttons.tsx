/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Tooltip } from "@plane/propel/tooltip";
import { cn } from "@plane/utils/common";
import type { ToolbarMenuItem } from "@/constants/common";

const STYLES = {
  comment: {
    group: "gap-0.5 border-r border-subtle px-2.5 first:pl-0",
    button: "text-placeholder hover:bg-layer-1",
    activeButton: "bg-layer-1 text-primary",
    activeIcon: "text-primary",
  },
  publicComment: {
    group: "gap-0.5 border-r border-subtle px-2.5 first:pl-0",
    button: "text-placeholder hover:bg-layer-transparent-hover",
    activeButton: "bg-layer-transparent-hover text-primary",
    activeIcon: "text-primary",
  },
  sticky: {
    group: "gap-4",
    button: "text-primary/50",
    activeButton: "",
    activeIcon: "font-heavy",
  },
};

type Props = {
  groups: Record<string, ToolbarMenuItem[]>;
  activeStates: Record<string, boolean>;
  executeCommand: (item: ToolbarMenuItem) => void;
  variant: keyof typeof STYLES;
};

export function EditorToolbarButtons({ groups, activeStates, executeCommand, variant }: Props) {
  const style = STYLES[variant];
  return Object.entries(groups).map(([key, items]) => (
    <div key={key} className={cn("flex items-stretch", style.group)}>
      {items.map((item) => {
        const active = activeStates[item.renderKey] ?? false;
        return (
          <Tooltip
            key={item.renderKey}
            tooltipContent={
              <p className="flex flex-col gap-1 text-center text-11">
                <span className="font-medium">{item.name}</span>
                {item.shortcut && <kbd className="text-placeholder">{item.shortcut.join(" + ")}</kbd>}
              </p>
            }
          >
            <button
              type="button"
              aria-label={item.name}
              aria-pressed={active}
              onClick={() => executeCommand(item)}
              className={cn(
                "grid aspect-square place-items-center rounded-xs p-0.5",
                style.button,
                active && style.activeButton
              )}
            >
              <item.icon className={cn("h-3.5 w-3.5", active && style.activeIcon)} strokeWidth={2.5} />
            </button>
          </Tooltip>
        );
      })}
    </div>
  ));
}
