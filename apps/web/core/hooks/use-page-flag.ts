/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export type TPageFlagHookArgs = {
  workspaceSlug: string;
};

export type TPageFlagHookReturnType = {
  isMovePageEnabled: boolean;
  isPageSharingEnabled: boolean;
};

export const usePageFlag = (_args: TPageFlagHookArgs): TPageFlagHookReturnType => {
  return {
    // These controls have no move modal or sharing implementation in this build.
    isMovePageEnabled: false,
    isPageSharingEnabled: false,
  };
};
