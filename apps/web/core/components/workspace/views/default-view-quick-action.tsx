/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTranslation } from "@plane/i18n";
// plane imports
import { LinkIcon, NewTabIcon } from "@plane/propel/icons";
import { useViewLinkCopy } from "@/components/views/use-view-link-copy";
// ui
import type { TStaticViewTypes } from "@plane/types";
import type { TContextMenuItem } from "@plane/ui";
import { CustomMenu } from "@plane/ui";
// helpers
type Props = {
  workspaceSlug: string;
  view: {
    key: TStaticViewTypes;
    i18n_label: string;
  };
};

export const DefaultWorkspaceViewQuickActions = function DefaultWorkspaceViewQuickActions(props: Props) {
  const { workspaceSlug, view } = props;

  const { t } = useTranslation();

  const viewLink = `${workspaceSlug}/workspace-views/${view.key}`;
  const handleCopyText = useViewLinkCopy(viewLink);
  const handleOpenInNewTab = () => window.open(`/${viewLink}`, "_blank");

  const MENU_ITEMS: TContextMenuItem[] = [
    {
      key: "open-new-tab",
      action: handleOpenInNewTab,
      title: t("open_in_new_tab"),
      icon: NewTabIcon,
    },
    {
      key: "copy-link",
      action: () => void handleCopyText(),
      title: t("copy_link"),
      icon: LinkIcon,
    },
  ];

  return (
    <>
      <CustomMenu
        ellipsis
        placement="bottom-end"
        closeOnSelect
        buttonClassName="flex-shrink-0 flex items-center justify-center size-[26px] bg-layer-1/70 rounded-sm"
      >
        {MENU_ITEMS.map((item) => (
          <CustomMenu.ActionItem key={item.key} item={item} iconClassName="h-3 w-3" />
        ))}
      </CustomMenu>
    </>
  );
};
