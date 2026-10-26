/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { PanelLeft, MoveRight } from "lucide-react";
import { IconButton } from "@plane/propel/icon-button";
import { ChevronDownIcon, ChevronUpIcon } from "@plane/propel/icons";
import type { TNameDescriptionLoader, TInboxIssue } from "@plane/types";
import { Header, EHeaderVariant } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { NameDescriptionUpdateStatus } from "@/components/issues/issue-update-status";
import { InboxIssueStatus } from "../inbox-issue-status";
import { IntakeMobileActionMenu } from "./intake-action-controls";
import type { IntakeActions } from "./intake-action-controls";

type Props = {
  inboxIssue: TInboxIssue;
  actions: IntakeActions;
  isSubmitting: TNameDescriptionLoader;
  handleInboxIssueNavigation: (direction: "next" | "prev") => void;
  isMobileSidebar: boolean;
  setIsMobileSidebar: (value: boolean) => void;
  isNotificationEmbed: boolean;
  embedRemoveCurrentNotification?: () => void;
};

export function InboxIssueActionsMobileHeader({
  inboxIssue,
  actions,
  isSubmitting,
  handleInboxIssueNavigation,
  isMobileSidebar,
  setIsMobileSidebar,
  isNotificationEmbed,
  embedRemoveCurrentNotification,
}: Props) {
  return (
    <Header variant={EHeaderVariant.SECONDARY} className="justify-start">
      {isNotificationEmbed && (
        <button onClick={embedRemoveCurrentNotification}>
          <MoveRight className="mr-2 h-4 w-4 text-tertiary hover:text-secondary" />
        </button>
      )}
      <PanelLeft
        onClick={() => setIsMobileSidebar(!isMobileSidebar)}
        className={cn("my-auto mr-2 h-4 w-4 flex-shrink-0", isMobileSidebar ? "text-accent-primary" : "text-secondary")}
      />
      <div className="z-[15] flex w-full items-center gap-2 bg-surface-1">
        <div className="flex items-center gap-x-2">
          <IconButton
            variant="secondary"
            size="lg"
            icon={ChevronUpIcon}
            aria-label="Previous work item"
            onClick={() => handleInboxIssueNavigation("prev")}
          />
          <IconButton
            variant="secondary"
            size="lg"
            icon={ChevronDownIcon}
            aria-label="Next work item"
            onClick={() => handleInboxIssueNavigation("next")}
          />
        </div>
        <div className="flex items-center gap-4">
          <InboxIssueStatus inboxIssue={inboxIssue} iconSize={12} />
          <div className="flex w-full items-center justify-end">
            <NameDescriptionUpdateStatus isSubmitting={isSubmitting} />
          </div>
        </div>
        <div className="ml-auto">
          <IntakeMobileActionMenu actions={actions} />
        </div>
      </div>
    </Header>
  );
}
