/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { EPageAccess } from "@plane/constants";
import type { TPage } from "@plane/types";
import type { CreatePayload, BaseActionPayload } from "@/types";

// Define all payload types for each event.
export type ArchivedPayload = CreatePayload<{ archived_at: string | null }>;
export type UnarchivedPayload = BaseActionPayload;
export type LockedPayload = CreatePayload<{ is_locked: boolean }>;
export type UnlockedPayload = BaseActionPayload;
export type MadePublicPayload = CreatePayload<{ access: EPageAccess }>;
export type MadePrivatePayload = CreatePayload<{ access: EPageAccess }>;
export type DeletedPayload = CreatePayload<{ deleted_at: Date | null }>;
export type DuplicatedPayload = CreatePayload<{ new_page_id: string }>;
export type PropertyUpdatedPayload = CreatePayload<Partial<TPage>>;
export type MovedPayload = CreatePayload<{
  new_project_id: string;
  new_page_id: string;
}>;
export type RestoredPayload = CreatePayload<{ deleted_page_ids?: string[] }>;
export type ErrorPayload = CreatePayload<{
  error_message: string;
  error_type: "fetch" | "store";
  error_code?: "content_too_large" | "page_locked" | "page_archived";
  should_disconnect?: boolean;
}>;

export type DocumentEventPayloads = {
  lock: LockedPayload;
  unlock: UnlockedPayload;
  archive: ArchivedPayload;
  unarchive: UnarchivedPayload;
  "make-public": MadePublicPayload;
  "make-private": MadePrivatePayload;
  delete: DeletedPayload;
  move: MovedPayload;
  duplicate: DuplicatedPayload;
  property_update: PropertyUpdatedPayload;
  restore: RestoredPayload;
  error: ErrorPayload;
};

export const DocumentCollaborativeEvents = {
  lock: {
    client: "locked",
    server: "lock",
  },
  unlock: {
    client: "unlocked",
    server: "unlock",
  },
  archive: {
    client: "archived",
    server: "archive",
  },
  unarchive: {
    client: "unarchived",
    server: "unarchive",
  },
  "make-public": {
    client: "made-public",
    server: "make-public",
  },
  "make-private": {
    client: "made-private",
    server: "make-private",
  },
  delete: {
    client: "deleted",
    server: "delete",
  },
  move: {
    client: "moved",
    server: "move",
  },
  duplicate: {
    client: "duplicated",
    server: "duplicate",
  },
  property_update: {
    client: "property_updated",
    server: "property_update",
  },
  restore: {
    client: "restored",
    server: "restore",
  },
  error: {
    client: "error",
    server: "error",
  },
} as const satisfies { [K in keyof DocumentEventPayloads]: { client: string; server: K } };
