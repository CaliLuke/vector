/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { InboxIssueConfirmationProps } from "./confirmation-types";
import { InboxIssueConfirmation } from "./issue-confirmation";

export function DeleteInboxIssueModal(props: InboxIssueConfirmationProps) {
  return <InboxIssueConfirmation {...props} action="delete" />;
}
