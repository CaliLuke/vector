/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TEditorHookProps } from "@/types";

export function createEditorHookProps(overrides: Partial<TEditorHookProps> = {}): TEditorHookProps {
  return {
    id: "test-editor",
    editable: true,
    enableHistory: true,
    value: null,
    disabledExtensions: [],
    flaggedExtensions: [],
    extendedEditorProps: {},
    fileHandler: {
      assetsUploadStatus: {},
      cancel: () => {},
      checkIfAssetExists: async () => false,
      delete: async () => {},
      getAssetDownloadSrc: async (path) => path,
      getAssetSrc: async (path) => path,
      restore: async () => {},
      upload: async () => "asset",
      duplicate: async (id) => id,
      validation: { maxFileSize: 1000 },
    },
    getEditorMetaData: () => ({ file_assets: [], user_mentions: [] }),
    mentionHandler: {
      searchCallback: async () => [],
      renderComponent: () => null,
    },
    ...overrides,
  };
}
