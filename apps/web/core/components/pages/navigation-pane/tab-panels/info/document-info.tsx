/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { usePageRecord } from "@/components/pages/page-record-context";

import { useCallback, useMemo, useSyncExternalStore } from "react";
// plane imports
import type { EditorRefApi, TDocumentInfo } from "@plane/editor";
import { useTranslation } from "@plane/i18n";
import { getReadTimeFromWordsCount } from "@plane/utils/datetime";
// store

const DEFAULT_DOCUMENT_INFO: TDocumentInfo = {
  words: 0,
  characters: 0,
  paragraphs: 0,
};

function documentInfoSubscription(editor: EditorRefApi | null) {
  let snapshot = editor?.getDocumentInfo() ?? DEFAULT_DOCUMENT_INFO;
  return {
    getSnapshot: () => snapshot,
    subscribe: (notify: () => void) => {
      const update = (next: TDocumentInfo) => {
        if (
          next.words === snapshot.words &&
          next.characters === snapshot.characters &&
          next.paragraphs === snapshot.paragraphs
        )
          return;
        snapshot = next;
        notify();
      };
      const unsubscribe = editor?.onDocumentInfoChange(update);
      update(editor?.getDocumentInfo() ?? DEFAULT_DOCUMENT_INFO);
      return () => unsubscribe?.();
    },
  };
}

export function PageNavigationPaneInfoTabDocumentInfo() {
  const { editorRef } = usePageRecord();
  const { t } = useTranslation();
  const subscription = useMemo(() => documentInfoSubscription(editorRef), [editorRef]);
  const documentInfo = useSyncExternalStore(
    subscription.subscribe,
    subscription.getSnapshot,
    () => DEFAULT_DOCUMENT_INFO
  );

  const secondsToReadableTime = useCallback(() => {
    const wordsCount = documentInfo.words;
    const readTimeInSeconds = Number(getReadTimeFromWordsCount(wordsCount).toFixed(0));
    return readTimeInSeconds < 60 ? `${readTimeInSeconds}s` : `${Math.ceil(readTimeInSeconds / 60)}m`;
  }, [documentInfo.words]);

  const documentInfoCards = useMemo(
    () => [
      {
        key: "words-count",
        title: t("page_navigation_pane.tabs.info.document_info.words"),
        info: documentInfo.words,
      },
      {
        key: "characters-count",
        title: t("page_navigation_pane.tabs.info.document_info.characters"),
        info: documentInfo.characters,
      },
      {
        key: "paragraphs-count",
        title: t("page_navigation_pane.tabs.info.document_info.paragraphs"),
        info: documentInfo.paragraphs,
      },
      {
        key: "read-time",
        title: t("page_navigation_pane.tabs.info.document_info.read_time"),
        info: secondsToReadableTime(),
      },
    ],
    [documentInfo, secondsToReadableTime, t]
  );

  return (
    <div className="grid grid-cols-2 gap-2">
      {documentInfoCards.map((card) => (
        <div key={card.key} className="rounded-sm bg-layer-2 p-2">
          <h6 className="text-14 font-semibold">{card.info}</h6>
          <p className="mt-1.5 text-13 font-medium text-tertiary">{card.title}</p>
        </div>
      ))}
    </div>
  );
}
