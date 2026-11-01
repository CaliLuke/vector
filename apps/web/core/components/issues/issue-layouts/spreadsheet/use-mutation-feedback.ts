/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useParams } from "@tanstack/react-router";
import { setToast, TOAST_TYPE } from "@plane/propel/toast";
import type { TIssue } from "@plane/types";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";

/** Cell actions own error feedback; late failures must not reach a replacement row or account. */
export function useSpreadsheetMutationFeedback(issue: TIssue) {
  const { workspaceSlug } = useParams({ strict: false });
  const accountId = useCurrentUserId();
  const client = useQueryClient();
  const generation = useIdentityGeneration(client);
  const current = useCommittedScope(JSON.stringify([accountId, generation, workspaceSlug, issue.project_id, issue.id]));
  return useCallback(() => {
    if (current())
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Could not update work item",
        message: "Please try again.",
      });
  }, [current]);
}
