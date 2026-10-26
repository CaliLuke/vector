/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import useReloadConfirmations from "./use-reload-confirmation";

type TEntityDetails = {
  entityID: string;
  groupID: string;
};

type Props = {
  containerRef: React.MutableRefObject<HTMLElement | null>;
  disabled: boolean;
  entities: Record<string, string[]>;
};

export type TSelectionHelper = {
  isSelectionActive: boolean;
  handleClearSelection: () => void;
  handleEntityClick: (event: React.MouseEvent, entityID: string, groupId: string) => void;
  getIsEntitySelected: (entityID: string) => boolean;
  getIsEntityActive: (entityID: string) => boolean;
  handleGroupClick: (groupID: string) => void;
  isGroupSelected: (groupID: string) => "empty" | "partial" | "complete";
  isSelectionDisabled: boolean;
};

type Selection = { entities: TEntityDetails[]; anchor: TEntityDetails | null };

function isSelectionKey(event: KeyboardEvent, container: HTMLElement | null) {
  if (
    event.defaultPrevented ||
    event.metaKey ||
    event.ctrlKey ||
    event.altKey ||
    !["ArrowUp", "ArrowDown"].includes(event.key)
  )
    return false;
  const target = event.target;
  if (
    target instanceof Element &&
    target.closest(
      'input:not([type="checkbox"]), textarea, select, [contenteditable]:not([contenteditable="false"]), [role="dialog"]'
    )
  )
    return false;
  return !(target instanceof Node) || target === document.body || !!container?.contains(target);
}

export const useMultipleSelect = ({ containerRef, disabled, entities }: Props): TSelectionHelper => {
  const [selection, setSelection] = useState<Selection>({ entities: [], anchor: null });
  const [activeEntity, setActiveEntity] = useState<TEntityDetails | null>(null);
  const entitiesList = useMemo(
    () => Object.entries(entities).flatMap(([groupID, ids]) => ids.map((entityID) => ({ entityID, groupID }))),
    [entities]
  );
  // Prune in the same render, before descendants observe a removed entity.
  const availableIds = useMemo(() => new Set(entitiesList.map(({ entityID }) => entityID)), [entitiesList]);
  const visibleSelection = selection.entities.filter(({ entityID }) => availableIds.has(entityID));
  if (visibleSelection.length !== selection.entities.length) {
    setSelection({ entities: visibleSelection, anchor: visibleSelection.at(-1) ?? null });
  }
  if (activeEntity && !availableIds.has(activeEntity.entityID)) setActiveEntity(null);
  const selectedIds = useMemo(() => new Set(selection.entities.map(({ entityID }) => entityID)), [selection.entities]);
  const getIsEntitySelected = useCallback((entityID: string) => selectedIds.has(entityID), [selectedIds]);
  const getIsEntityActive = useCallback((entityID: string) => activeEntity?.entityID === entityID, [activeEntity]);
  const handleClearSelection = useCallback(() => {
    setSelection({ entities: [], anchor: null });
    setActiveEntity(null);
  }, []);

  useReloadConfirmations(
    !disabled && selectedIds.size > 0,
    "Are you sure you want to leave? Your current bulk operation selections will be lost.",
    true,
    handleClearSelection
  );

  const activate = useCallback(
    (entity: TEntityDetails, scroll: boolean) => {
      setActiveEntity(entity);
      const container = containerRef.current;
      if (!scroll || !container) return;
      const element = Array.from(container.querySelectorAll<HTMLElement>("[data-entity-id]")).find(
        (node) => node.dataset.entityId === entity.entityID && node.dataset.entityGroupId === entity.groupID
      );
      if (!element) return;
      const bounds = container.getBoundingClientRect();
      const itemBounds = element.getBoundingClientRect();
      const offset = 200;
      if (itemBounds.top < bounds.top + offset || itemBounds.bottom > bounds.bottom - offset) {
        container.scrollBy({ top: itemBounds.top < bounds.top + offset ? -50 : 50 });
      }
    },
    [containerRef]
  );

  const toggle = useCallback(
    (entity: TEntityDetails, scroll = false) => {
      setSelection((previous) => {
        const selected = previous.entities.some(({ entityID }) => entityID === entity.entityID);
        const remaining = previous.entities.filter(({ entityID }) => entityID !== entity.entityID);
        const next = selected ? remaining : [...remaining, entity];
        return { entities: next, anchor: selected ? (next.at(-1) ?? null) : entity };
      });
      activate(entity, scroll);
    },
    [activate]
  );

  const handleEntityClick = useCallback(
    (event: React.MouseEvent, entityID: string, groupID: string) => {
      if (disabled) return;
      const currentIndex = entitiesList.findIndex(
        (entity) => entity.entityID === entityID && entity.groupID === groupID
      );
      const anchorIndex = selection.anchor
        ? entitiesList.findIndex(
            (entity) => entity.entityID === selection.anchor?.entityID && entity.groupID === selection.anchor?.groupID
          )
        : -1;
      if (event.shiftKey && anchorIndex >= 0 && anchorIndex === currentIndex) return;
      if (event.shiftKey && anchorIndex >= 0 && currentIndex >= 0 && anchorIndex !== currentIndex) {
        const range =
          anchorIndex < currentIndex
            ? entitiesList.slice(anchorIndex + 1, currentIndex + 1)
            : entitiesList.slice(currentIndex, anchorIndex);
        range.forEach((entity) => toggle(entity));
        return;
      }
      toggle({ entityID, groupID });
    },
    [disabled, entitiesList, selection.anchor, toggle]
  );

  const isGroupSelected = useCallback(
    (groupID: string) => {
      const ids = entities[groupID] ?? [];
      const count = ids.filter((id) => selectedIds.has(id)).length;
      if (count === 0) return "empty";
      return count === ids.length ? "complete" : "partial";
    },
    [entities, selectedIds]
  );

  const handleGroupClick = useCallback(
    (groupID: string) => {
      if (disabled) return;
      const members = entitiesList.filter((entity) => entity.groupID === groupID);
      if (!members.length) return;
      const ids = new Set(members.map(({ entityID }) => entityID));
      const add = isGroupSelected(groupID) === "empty";
      setSelection((previous) => {
        const remaining = previous.entities.filter(({ entityID }) => !ids.has(entityID));
        const next = add ? [...remaining, ...members] : remaining;
        return { entities: next, anchor: next.at(-1) ?? null };
      });
      if (add) activate(members[members.length - 1], false);
    },
    [activate, disabled, entitiesList, isGroupSelected]
  );

  useEffect(() => {
    if (disabled) return undefined;
    const onKeyDown = (event: KeyboardEvent) => {
      if (!isSelectionKey(event, containerRef.current)) return;
      const index = activeEntity
        ? entitiesList.findIndex(
            (entity) => entity.entityID === activeEntity.entityID && entity.groupID === activeEntity.groupID
          )
        : -1;
      if (event.shiftKey && index < 0) return;
      const next = index < 0 ? entitiesList[0] : entitiesList[index + (event.key === "ArrowDown" ? 1 : -1)];
      if (!next) return;
      event.preventDefault();
      if (event.shiftKey) toggle(next, true);
      else activate(next, true);
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [activate, activeEntity, containerRef, disabled, entitiesList, toggle]);

  return useMemo(
    () => ({
      isSelectionActive: selectedIds.size > 0,
      handleClearSelection,
      handleEntityClick,
      getIsEntitySelected,
      getIsEntityActive,
      handleGroupClick,
      isGroupSelected,
      isSelectionDisabled: disabled,
    }),
    [
      disabled,
      getIsEntityActive,
      getIsEntitySelected,
      handleClearSelection,
      handleEntityClick,
      handleGroupClick,
      isGroupSelected,
      selectedIds.size,
    ]
  );
};
