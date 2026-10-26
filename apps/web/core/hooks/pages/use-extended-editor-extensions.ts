/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TPage } from "@plane/types";

import type { IEditorPropsExtended } from "@plane/editor";
import type { TSearchEntityRequestPayload, TSearchResponse } from "@plane/types";

export type TExtendedEditorExtensionsHookParams = {
  workspaceSlug: string;
  page: TPage;
  fetchEntity: (payload: TSearchEntityRequestPayload) => Promise<TSearchResponse>;
  getRedirectionLink: (pageId?: string) => string;
  extensionHandlers?: Map<string, unknown>;
  projectId?: string;
};

export type TExtendedEditorExtensionsConfig = IEditorPropsExtended;

export const useExtendedEditorProps = (
  _params: TExtendedEditorExtensionsHookParams
): TExtendedEditorExtensionsConfig => ({});
