/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// helpers
import type { IWebhook, IWorkspace } from "@plane/types";
import { renderFormattedPayloadDate } from "@plane/utils/datetime";
// types

const csvValue = (value: string | boolean | null | undefined): string => (value == null ? "" : String(value));

export const getCurrentHookAsCSV = (
  currentWorkspace: IWorkspace | null,
  webhook: IWebhook | undefined,
  secretKey: string | undefined
) => ({
  id: csvValue(webhook?.id),
  url: csvValue(webhook?.url),
  created_at: csvValue(renderFormattedPayloadDate(csvValue(webhook?.created_at))),
  updated_at: csvValue(renderFormattedPayloadDate(csvValue(webhook?.updated_at))),
  is_active: csvValue(webhook?.is_active),
  secret_key: csvValue(secretKey),
  project: csvValue(webhook?.project),
  issue: csvValue(webhook?.issue),
  module: csvValue(webhook?.module),
  cycle: csvValue(webhook?.cycle),
  issue_comment: csvValue(webhook?.issue_comment),
  workspace: csvValue(currentWorkspace?.name),
});
