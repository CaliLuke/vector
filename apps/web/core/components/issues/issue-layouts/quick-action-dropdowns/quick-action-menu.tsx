/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ComponentProps, RefObject } from "react";
import { ContextMenu, CustomMenu } from "@plane/ui";
import type { TContextMenuItem } from "@plane/ui";
import { cn } from "@plane/utils";

type QuickActionMenuProps = {
  items: TContextMenuItem[];
  parentRef: RefObject<HTMLElement | null>;
  customMenuProps: Omit<ComponentProps<typeof CustomMenu>, "children">;
};

type MenuItemProps = {
  item: TContextMenuItem;
};

function MenuItemIcon({ item }: MenuItemProps) {
  const Icon = item.icon;
  return Icon ? <Icon className={cn("h-3 w-3", item.iconClassName)} /> : null;
}

function MenuItemLabel({ item }: MenuItemProps) {
  return (
    <>
      <h5>{item.title}</h5>
      {item.description && (
        <p
          className={cn("whitespace-pre-line text-tertiary", {
            "text-placeholder": item.disabled,
          })}
        >
          {item.description}
        </p>
      )}
    </>
  );
}

function renderActionItem(item: TContextMenuItem) {
  return (
    <CustomMenu.MenuItem
      key={item.key}
      onClick={() => item.action()}
      className={cn(
        "flex items-center gap-2",
        {
          "text-placeholder": item.disabled,
        },
        item.className
      )}
      disabled={item.disabled}
    >
      <MenuItemIcon item={item} />
      <div>
        <MenuItemLabel item={item} />
      </div>
    </CustomMenu.MenuItem>
  );
}

function renderMenuItem(item: TContextMenuItem) {
  if (item.shouldRender === false) return null;

  if (item.nestedMenuItems && item.nestedMenuItems.length > 0) {
    return (
      <CustomMenu.SubMenu
        key={item.key}
        trigger={
          <div className="flex items-center gap-2">
            <MenuItemIcon item={item} />
            <MenuItemLabel item={item} />
          </div>
        }
        disabled={item.disabled}
        className={cn(
          "flex items-center gap-2",
          {
            "text-placeholder": item.disabled,
          },
          item.className
        )}
      >
        {item.nestedMenuItems.map(renderActionItem)}
      </CustomMenu.SubMenu>
    );
  }

  return renderActionItem(item);
}

export function QuickActionMenu(props: QuickActionMenuProps) {
  const { items, parentRef, customMenuProps } = props;

  return (
    <>
      <ContextMenu parentRef={parentRef} items={items} />
      <CustomMenu {...customMenuProps}>{items.map(renderMenuItem)}</CustomMenu>
    </>
  );
}
