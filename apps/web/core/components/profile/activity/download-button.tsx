/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";

// services
// ui
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { setToast, TOAST_TYPE } from "@plane/propel/toast";
// helpers
import { renderFormattedPayloadDate } from "@plane/utils/datetime";
import { UserService } from "@/services/user.service";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";

const userService = new UserService();

export function DownloadActivityButton() {
  const { workspaceSlug, userId } = useParams({ strict: false });
  const accountId = useCurrentUserId();
  const client = useQueryClient();
  const generation = useIdentityGeneration(client);
  const scope = JSON.stringify([accountId, generation, workspaceSlug, userId]);
  return <ActivityDownload key={scope} scope={scope} workspaceSlug={workspaceSlug} userId={userId} />;
}

function ActivityDownload({
  scope,
  workspaceSlug,
  userId,
}: {
  scope: string;
  workspaceSlug?: string;
  userId?: string;
}) {
  // states
  const [isDownloading, setIsDownloading] = useState(false);
  const current = useCommittedScope(scope);
  //hooks
  const { t } = useTranslation();

  const handleDownload = async () => {
    const today = renderFormattedPayloadDate(new Date());

    if (isDownloading || !workspaceSlug || !userId || !today) return;

    setIsDownloading(true);

    try {
      const csv = await userService.downloadProfileActivity(workspaceSlug, userId, { date: today });
      if (current()) downloadActivityCSV(csv);
    } catch {
      if (current())
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Could not download activity",
          message: "Please try again.",
        });
    } finally {
      if (current()) setIsDownloading(false);
    }
  };

  return (
    <Button onClick={() => void handleDownload()} loading={isDownloading}>
      {isDownloading ? t("profile.stats.recent_activity.button_loading") : t("profile.stats.recent_activity.button")}
    </Button>
  );
}

function downloadActivityCSV(csv: string) {
  const url = window.URL.createObjectURL(new Blob([csv], { type: "text/csv" }));
  const link = document.createElement("a");
  try {
    link.href = url;
    link.download = `profile-activity-${Date.now()}.csv`;
    document.body.appendChild(link);
    link.click();
  } finally {
    link.remove();
    window.URL.revokeObjectURL(url);
  }
}
