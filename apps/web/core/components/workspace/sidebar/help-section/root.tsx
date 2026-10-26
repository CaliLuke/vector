/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useState } from "react";
import { HelpCircle, User } from "lucide-react";
import { useTranslation } from "@plane/i18n";
import { PageIcon } from "@plane/propel/icons";
// ui
import { CustomMenu } from "@plane/ui";
// components
import { ProductUpdatesModal } from "@/components/global";
import { AppSidebarItem } from "@/components/sidebar/sidebar-item";
import { PlaneVersionNumber } from "@/components/global/version-number";
// hooks
import { usePowerK } from "@/hooks/use-power-k";

export function HelpMenuRoot() {
  // store hooks
  const { t } = useTranslation();
  const { toggleShortcutsListModal } = usePowerK();
  // states
  const [isNeedHelpOpen, setIsNeedHelpOpen] = useState(false);
  const [isProductUpdatesModalOpen, setProductUpdatesModalOpen] = useState(false);

  return (
    <>
      <ProductUpdatesModal isOpen={isProductUpdatesModalOpen} handleClose={() => setProductUpdatesModalOpen(false)} />

      <CustomMenu
        ariaLabel={t("help")}
        customButton={
          <AppSidebarItem
            variant="content"
            item={{
              icon: <HelpCircle className="size-5" />,
              isActive: isNeedHelpOpen,
            }}
          />
        }
        customButtonClassName="cursor-pointer appearance-none border-0 bg-transparent p-0 text-inherit outline-none"
        menuButtonOnClick={() => !isNeedHelpOpen && setIsNeedHelpOpen(true)}
        onMenuClose={() => setIsNeedHelpOpen(false)}
        placement="bottom-end"
        maxHeight="lg"
        closeOnSelect
      >
        <CustomMenu.MenuItem
          onClick={() => window.open("https://github.com/CaliLuke/vector/blob/main/docs/README.md", "_blank")}
        >
          <div className="flex items-center gap-x-2 rounded-sm text-11">
            <PageIcon className="h-3.5 w-3.5 text-secondary" height={14} width={14} />
            <span className="text-11">{t("documentation")}</span>
          </div>
        </CustomMenu.MenuItem>
        <CustomMenu.MenuItem onClick={() => window.open("https://github.com/CaliLuke/vector/issues", "_blank")}>
          <div className="flex items-center gap-x-2 rounded-sm text-11">
            <User className="h-3.5 w-3.5 text-secondary" size={14} />
            <span className="text-11">Vector issues</span>
          </div>
        </CustomMenu.MenuItem>
        <div className="my-1 border-t border-subtle" />
        <CustomMenu.MenuItem onClick={() => toggleShortcutsListModal(true)}>
          <span className="text-11">{t("keyboard_shortcuts")}</span>
        </CustomMenu.MenuItem>
        <CustomMenu.MenuItem onClick={() => setProductUpdatesModalOpen(true)}>
          <span className="text-11">{t("whats_new")}</span>
        </CustomMenu.MenuItem>
        <CustomMenu.MenuItem
          onClick={() =>
            window.open("https://github.com/CaliLuke/vector/blob/main/docs/README.md", "_blank", "noopener,noreferrer")
          }
        >
          <div className="flex items-center gap-x-2 rounded-sm text-11">
            <span className="text-11">Source and license</span>
          </div>
        </CustomMenu.MenuItem>
        <div className="mt-1 border-t border-subtle px-1 pt-2 text-11 text-secondary">
          <PlaneVersionNumber />
        </div>
      </CustomMenu>
    </>
  );
}
