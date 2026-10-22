/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { observer } from "mobx-react";
import { PlusIcon } from "@plane/propel/icons";
// plane imports
import type { TIssueServiceType } from "@plane/types";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
// components
import { IssueDetailWidgetButton } from "../widget-button";

type Props = {
  customButton?: React.ReactNode;
  disabled?: boolean;
  issueServiceType: TIssueServiceType;
};

export const IssueLinksActionButton = observer(function IssueLinksActionButton(props: Props) {
  const { customButton, disabled = false, issueServiceType } = props;
  // store hooks
  const { toggleIssueLinkModal } = useIssueDetail(issueServiceType);

  // handlers
  const handleOnClick = (e: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
    e.preventDefault();
    e.stopPropagation();
    toggleIssueLinkModal(true);
  };

  if (
    React.isValidElement<{ disabled?: boolean; onClick?: (event: React.MouseEvent<HTMLButtonElement>) => void }>(
      customButton
    ) &&
    (customButton.type === "button" || customButton.type === IssueDetailWidgetButton)
  ) {
    const customButtonOnClick = customButton.props.onClick;
    return React.cloneElement(customButton, {
      disabled: disabled || customButton.props.disabled,
      onClick: (event) => {
        customButtonOnClick?.(event);
        if (!event.defaultPrevented) handleOnClick(event);
      },
    });
  }

  return (
    <button type="button" onClick={handleOnClick} disabled={disabled}>
      {customButton ?? <PlusIcon className="h-4 w-4" />}
    </button>
  );
});
