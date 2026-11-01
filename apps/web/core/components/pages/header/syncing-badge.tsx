/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { CloudOff, Dot } from "lucide-react";
import { Tooltip } from "@plane/propel/tooltip";
import { Badge } from "@plane/propel/badge";

type Props = {
  syncStatus: "syncing" | "synced" | "error";
};

export function PageSyncingBadge({ syncStatus }: Props) {
  if (syncStatus === "synced") return null;

  const badgeContent = {
    syncing: {
      label: "Syncing...",
      tooltipHeading: "Syncing...",
      tooltipContent: "Your changes are being synced with the server. You can continue making changes.",
    },
    error: {
      label: "Connection lost",
      tooltipHeading: "Connection lost",
      tooltipContent:
        "We're having trouble connecting to the websocket server. Your changes will be synced and saved every 30 seconds.",
    },
  };

  // This way we guarantee badgeContent is defined
  const content = badgeContent[syncStatus];

  return (
    <Tooltip tooltipHeading={content.tooltipHeading} tooltipContent={content.tooltipContent}>
      <span className="animate-quickFadeIn">
        <Badge
          variant={syncStatus === "syncing" ? "brand" : "danger"}
          size="lg"
          prependIcon={syncStatus === "syncing" ? <Dot /> : <CloudOff />}
        >
          {content.label}
        </Badge>
      </span>
    </Tooltip>
  );
}
