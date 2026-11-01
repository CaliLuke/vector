/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { renderFormattedDate as formatDate } from "@plane/utils/datetime";

export { calculateTimeAgo as timeAgo } from "@plane/utils/datetime";

// Space's existing presentation contract uses null for an absent or invalid date.
export const renderFormattedDate = (date: string | Date | undefined | null): string | null => formatDate(date) ?? null;
