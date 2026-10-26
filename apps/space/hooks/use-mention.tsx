/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQueryClient } from "@tanstack/react-query";
import { currentUserOptions } from "@/queries/session";
export const useMention = () => {
  const queryClient = useQueryClient();

  const mentionHighlights = async (): Promise<string[]> => {
    try {
      const user = await queryClient.ensureQueryData(currentUserOptions);
      return user ? [user.id] : [];
    } catch {
      // Anonymous readers have no personal mentions to highlight.
      return [];
    }
  };

  return { mentionHighlights };
};
