/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { LogOut } from "lucide-react";
import { MEMBER_TRACKER_ELEMENTS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { CustomMenu } from "@plane/ui";

export function LeaveProjectMenuItem({ onLeave }: { onLeave: () => void }) {
  const { t } = useTranslation();
  return (
    <CustomMenu.MenuItem onClick={onLeave} data-ph-element={MEMBER_TRACKER_ELEMENTS.SIDEBAR_PROJECT_QUICK_ACTIONS}>
      <div className="flex items-center justify-start gap-2">
        <LogOut className="h-3.5 w-3.5 stroke-[1.5]" />
        <span>{t("leave_project")}</span>
      </div>
    </CustomMenu.MenuItem>
  );
}
