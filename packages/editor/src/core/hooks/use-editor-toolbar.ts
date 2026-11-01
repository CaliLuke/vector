/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo, useSyncExternalStore } from "react";
import type { ToolbarMenuItem } from "@/constants/common";
import type { EditorRefApi } from "@/types";

type ToolbarEditor = Pick<EditorRefApi, "onStateChange" | "isMenuItemActive" | "isEditorReadyToDiscard">;
type ToolbarSnapshot = { activeStates: Record<string, boolean>; isReadyToDiscard: boolean };
const EMPTY_SNAPSHOT: ToolbarSnapshot = { activeStates: {}, isReadyToDiscard: false };
const getServerSnapshot = () => EMPTY_SNAPSHOT;

function createToolbarStore(editor: ToolbarEditor | null, groups: Record<string, ToolbarMenuItem[]>) {
  const items = Object.values(groups).flat();
  const readSnapshot = (): ToolbarSnapshot => ({
    activeStates: Object.fromEntries(
      items.map((item) => [
        item.renderKey,
        editor?.isMenuItemActive({ itemKey: item.itemKey, ...item.extraProps }) ?? false,
      ])
    ),
    isReadyToDiscard: editor?.isEditorReadyToDiscard() ?? false,
  });
  let snapshot = readSnapshot();
  return {
    getSnapshot: () => snapshot,
    subscribe: (notify: () => void) => {
      const update = () => {
        const next = readSnapshot();
        if (
          next.isReadyToDiscard === snapshot.isReadyToDiscard &&
          items.every((item) => next.activeStates[item.renderKey] === snapshot.activeStates[item.renderKey])
        )
          return;
        snapshot = next;
        notify();
      };
      const unsubscribe = editor?.onStateChange(update);
      update();
      return () => unsubscribe?.();
    },
  };
}

export function useEditorToolbar(editor: ToolbarEditor | null, groups: Record<string, ToolbarMenuItem[]>) {
  const store = useMemo(() => createToolbarStore(editor, groups), [editor, groups]);
  return useSyncExternalStore(store.subscribe, store.getSnapshot, getServerSnapshot);
}
