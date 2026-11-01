/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { DocumentCollaborativeEvents } from "@/constants/document-collaborative-events";
import type { TDocumentEventsClient, TDocumentEventsServer } from "@/types/document-collaborative-events";

export const getServerEventName = (clientEvent: TDocumentEventsClient): TDocumentEventsServer | undefined => {
  return Object.values(DocumentCollaborativeEvents).find((event) => event.client === clientEvent)?.server;
};
