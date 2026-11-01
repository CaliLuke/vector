/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Node as ProseMirrorNode } from "@tiptap/pm/model";
// plane imports
import type { ADDITIONAL_EXTENSIONS } from "@plane/utils/editor/common";
import { CORE_EXTENSIONS } from "@plane/utils/editor/common";
// extensions
import { getImageBlockId } from "@/extensions/custom-image/utils";
// plane editor imports
import { ADDITIONAL_ASSETS_META_DATA_RECORD } from "@/plane-editor/constants/assets";
// types
import type { TEditorAsset } from "@/types";

export type TAssetMetaDataRecord = (attrs: ProseMirrorNode["attrs"]) => TEditorAsset | undefined;

function imageAsset(
  attrs: ProseMirrorNode["attrs"],
  type: typeof CORE_EXTENSIONS.IMAGE | typeof CORE_EXTENSIONS.CUSTOM_IMAGE
) {
  const { src, id } = attrs;
  if (typeof src !== "string" || !src || typeof id !== "string" || !id) return undefined;
  return { href: `#${getImageBlockId(id)}`, id, name: `image-${id}`, size: 0, src, type };
}

export const CORE_ASSETS_META_DATA_RECORD: Partial<
  Record<CORE_EXTENSIONS | ADDITIONAL_EXTENSIONS, TAssetMetaDataRecord>
> = {
  [CORE_EXTENSIONS.IMAGE]: (attrs) => imageAsset(attrs, CORE_EXTENSIONS.IMAGE),
  [CORE_EXTENSIONS.CUSTOM_IMAGE]: (attrs) => imageAsset(attrs, CORE_EXTENSIONS.CUSTOM_IMAGE),
  ...ADDITIONAL_ASSETS_META_DATA_RECORD,
};
