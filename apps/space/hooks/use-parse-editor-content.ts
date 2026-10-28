/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
import { useCallback } from "react";
// helpers
import type { TCustomComponentsMetaData } from "@plane/utils/editor/markdown-parser";
// helpers
import { getEditorAssetSrc } from "@/helpers/editor.helper";
// hooks

import { membersOptions } from "@/queries/lookups";

type TArgs = {
  anchor: string;
};

export const useParseEditorContent = (args: TArgs) => {
  const { anchor } = args;
  // store hooks
  const { data: members = [] } = useRecordCollection(membersOptions(anchor));

  const getEditorMetaData = useCallback(
    (htmlContent: string): TCustomComponentsMetaData => {
      const document = new DOMParser().parseFromString(htmlContent, "text/html");
      const file_assets = Array.from(document.querySelectorAll("image-component")).flatMap((element) => {
        const src = element.getAttribute("src");
        if (!src) return [];
        const url = src.startsWith("http") ? src : getEditorAssetSrc(anchor, src);
        return url ? [{ id: src, name: src, url }] : [];
      });
      const user_mentions = Array.from(document.querySelectorAll("mention-component")).flatMap((element) => {
        const id = element.getAttribute("entity_identifier");
        const member = members.find((item) => item.member === id);
        if (!id || !member) return [];
        return [{ id, display_name: member.member__display_name, url: `${window.location.origin}/profile/${id}` }];
      });
      return { file_assets, user_mentions };
    },
    [anchor, members]
  );

  return {
    getEditorMetaData,
  };
};
