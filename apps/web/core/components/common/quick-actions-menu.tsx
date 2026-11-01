/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { RefObject } from "react";
import { MoreHorizontal } from "lucide-react";
import { IconButton } from "@plane/propel/icon-button";
import type { TContextMenuItem } from "@plane/ui";
import { ContextMenu, CustomMenu } from "@plane/ui";

type Props = {
  parentRef: RefObject<HTMLElement | null>;
  items: TContextMenuItem[];
  className?: string;
  maxHeight?: "lg";
};

export function QuickActionsMenu({ parentRef, items, className, maxHeight }: Props) {
  return (
    <>
      <ContextMenu parentRef={parentRef} items={items} />
      <CustomMenu
        customButtonAsChild
        customButton={<IconButton variant="tertiary" size="lg" icon={MoreHorizontal} aria-label="More actions" />}
        placement="bottom-end"
        closeOnSelect
        buttonClassName={className}
        maxHeight={maxHeight}
      >
        {items.map((item) => (
          <CustomMenu.ActionItem key={item.key} item={item} />
        ))}
      </CustomMenu>
    </>
  );
}
