/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useMemo } from "react";
// plane utils
import { calculateTimeAgo } from "@plane/utils/datetime";
import { cn } from "@plane/utils/common";
import { getIconForLink } from "@plane/utils/get-icon-for-link";
// plane ui
import type { TContextMenuItem } from "../dropdowns/context-menu/root";
import { CustomMenu } from "../dropdowns/custom-menu";

export type TLinkItemBlockProps = {
  title: string;
  url: string;
  createdAt?: Date | string;
  menuItems?: TContextMenuItem[];
  onClick?: () => void;
};

export function LinkItemBlock(props: TLinkItemBlockProps) {
  // props
  const { title, url, createdAt, menuItems, onClick } = props;
  // icons
  const icon = useMemo(
    () =>
      React.createElement(getIconForLink(url), {
        className: "size-4 stroke-2 text-tertiary group-hover:text-primary",
      }),
    [url]
  );
  const content = (
    <>
      <span className="grid size-8 flex-shrink-0 place-items-center rounded-sm bg-surface-2 p-2">{icon}</span>
      <span className="min-w-0 flex-1 truncate">
        <span className="block truncate text-13 font-medium">{title}</span>
        {createdAt && <span className="block text-11 font-medium text-placeholder">{calculateTimeAgo(createdAt)}</span>}
      </span>
    </>
  );
  const actionClassName =
    "flex min-w-0 flex-1 items-center gap-4 self-stretch text-left focus-visible:outline-2 focus-visible:outline-accent-strong";
  return (
    <div className="group flex h-[56px] w-[230px] items-center gap-4 rounded-md border-[0.5px] border-subtle bg-surface-1 px-4">
      {onClick ? (
        <button type="button" className={actionClassName} onClick={onClick}>
          {content}
        </button>
      ) : (
        <a href={url} target="_blank" rel="noopener noreferrer" className={actionClassName}>
          {content}
        </a>
      )}
      {menuItems && (
        <div className="invisible group-focus-within:visible group-hover:visible">
          <CustomMenu
            ariaLabel="Link actions"
            placement="bottom-end"
            menuItemsClassName="z-20"
            closeOnSelect
            verticalEllipsis
          >
            {menuItems.map((item) => (
              <CustomMenu.MenuItem
                key={item.key}
                onClick={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  item.action();
                }}
                className={cn("flex w-full items-center gap-2", {
                  "text-placeholder": item.disabled,
                })}
                disabled={item.disabled}
              >
                {item.icon && <item.icon className={cn("h-3 w-3", item.iconClassName)} />}
                <div>
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
                </div>
              </CustomMenu.MenuItem>
            ))}
          </CustomMenu>
        </div>
      )}
    </div>
  );
}
