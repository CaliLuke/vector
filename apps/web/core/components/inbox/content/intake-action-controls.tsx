/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Clock, FileStack, MoreHorizontal } from "lucide-react";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { getIconButtonStyling } from "@plane/propel/icon-button";
import {
  LinkIcon,
  CopyIcon,
  NewTabIcon,
  TrashIcon,
  CheckCircleFilledIcon,
  CloseCircleFilledIcon,
} from "@plane/propel/icons";
import { ControlLink } from "@plane/ui/control-link";
import { CustomMenu } from "@plane/ui/dropdowns";
import type { TContextMenuItem } from "@plane/ui/dropdowns";
import { renderQuickActionMenuItems } from "@/components/common/quick-action-menu";

export type IntakeActions = {
  canTriage: boolean;
  canDelete: boolean;
  isAllowed: boolean;
  isClosed: boolean;
  isSnoozed: boolean;
  workItemLink: string;
  accept: () => void;
  decline: () => void;
  snooze: () => void;
  duplicate: () => void;
  remove: () => void;
  copy: () => Promise<void>;
  open: () => void;
};

function useIntakeActionItems(actions: IntakeActions) {
  const { t } = useTranslation();
  const triage: TContextMenuItem[] = [
    {
      key: "accept",
      title: t("inbox_issue.actions.accept"),
      icon: CheckCircleFilledIcon,
      iconClassName: "text-success-secondary",
      action: actions.accept,
      shouldRender: actions.canTriage,
    },
    {
      key: "decline",
      title: t("inbox_issue.actions.decline"),
      icon: CloseCircleFilledIcon,
      iconClassName: "text-danger-secondary",
      action: actions.decline,
      shouldRender: actions.canTriage,
    },
  ];
  const pending: TContextMenuItem[] = [
    {
      key: "snooze",
      title: t(actions.isSnoozed ? "inbox_issue.actions.unsnooze" : "inbox_issue.actions.snooze"),
      icon: Clock,
      action: actions.snooze,
      shouldRender: actions.canTriage,
    },
    {
      key: "duplicate",
      title: t("inbox_issue.actions.mark_as_duplicate"),
      icon: FileStack,
      action: actions.duplicate,
      shouldRender: actions.canTriage,
    },
  ];
  const copy: TContextMenuItem = {
    key: "copy",
    title: t("inbox_issue.actions.copy"),
    icon: CopyIcon,
    action: () => {
      void actions.copy();
    },
  };
  const remove: TContextMenuItem = {
    key: "delete",
    title: t("inbox_issue.actions.delete"),
    icon: TrashIcon,
    action: actions.remove,
    shouldRender: actions.canDelete,
  };
  const open: TContextMenuItem = {
    key: "open",
    title: t("inbox_issue.actions.open"),
    icon: NewTabIcon,
    action: actions.open,
  };
  return { triage, pending, copy, remove, open };
}

function IntakeMenu({ items }: { items: TContextMenuItem[] }) {
  return (
    <CustomMenu
      customButton={<MoreHorizontal className="size-4" />}
      customButtonClassName={getIconButtonStyling("secondary", "lg")}
      placement="bottom-start"
    >
      {renderQuickActionMenuItems(items)}
    </CustomMenu>
  );
}

export function IntakeMobileActionMenu({ actions }: { actions: IntakeActions }) {
  const items = useIntakeActionItems(actions);
  return (
    <IntakeMenu
      items={
        actions.isClosed ? [items.copy, items.open, ...items.triage] : [...items.pending, ...items.triage, items.remove]
      }
    />
  );
}

export function IntakeDesktopActions({ actions }: { actions: IntakeActions }) {
  const { t } = useTranslation();
  const items = useIntakeActionItems(actions);
  return (
    <div className="flex flex-wrap items-center gap-2">
      {actions.canTriage && (
        <>
          <Button variant="secondary" size="lg" onClick={actions.accept}>
            <CheckCircleFilledIcon className="size-4 shrink-0 text-success-secondary" />
            {t("inbox_issue.actions.accept")}
          </Button>
          <Button variant="secondary" size="lg" onClick={actions.decline}>
            <CloseCircleFilledIcon className="size-4 shrink-0 text-danger-secondary" />
            {t("inbox_issue.actions.decline")}
          </Button>
        </>
      )}
      {actions.isClosed ? (
        <div className="flex items-center gap-2">
          <Button
            variant="secondary"
            size="lg"
            prependIcon={<LinkIcon className="h-2.5 w-2.5" />}
            onClick={() => void actions.copy()}
          >
            {t("inbox_issue.actions.copy")}
          </Button>
          <ControlLink href={actions.workItemLink} onClick={actions.open} target="_self">
            <Button variant="secondary" size="lg" prependIcon={<NewTabIcon className="h-2.5 w-2.5" />}>
              {t("inbox_issue.actions.open")}
            </Button>
          </ControlLink>
        </div>
      ) : (
        actions.isAllowed && <IntakeMenu items={[...items.pending, items.copy, items.remove]} />
      )}
    </div>
  );
}
