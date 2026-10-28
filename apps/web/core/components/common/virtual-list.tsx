import { useCallback, useState } from "react";
import type { ReactNode, RefObject } from "react";
import { defaultRangeExtractor, useVirtualizer } from "@tanstack/react-virtual";
import type { Range } from "@tanstack/react-virtual";

type Props<T> = {
  items: readonly T[];
  getKey: (item: T) => string;
  renderItem: (item: T) => ReactNode;
  scrollRef: RefObject<HTMLDivElement | null>;
  estimateSize: number;
  label: string;
};

/** Keep the focused row mounted while scrolling, including its portaled menus. */
export function VirtualList<T>({ items, getKey, renderItem, scrollRef, estimateSize, label }: Props<T>) {
  "use no memo"; // https://github.com/TanStack/virtual/issues/736 — remove when Virtual supports React Compiler.
  const itemKey = useCallback((index: number) => getKey(items[index]), [getKey, items]);
  const size = useCallback(() => estimateSize, [estimateSize]);
  const [focusedKey, setFocusedKey] = useState<string>();
  const focusedIndex = items.findIndex((item) => getKey(item) === focusedKey);
  const rangeExtractor = useCallback(
    (range: Range) => {
      const indexes = defaultRangeExtractor(range);
      if (focusedIndex < 0 || indexes.includes(focusedIndex)) return indexes;
      return focusedIndex < indexes[0] ? [focusedIndex, ...indexes] : [...indexes, focusedIndex];
    },
    [focusedIndex]
  );
  // The explicit compiler opt-out above is required; the rule still reports it (#1119).
  // oxlint-disable-next-line react/incompatible-library -- https://github.com/TanStack/virtual/issues/1119
  const virtualizer = useVirtualizer({
    count: items.length,
    getScrollElement: () => scrollRef.current,
    getItemKey: itemKey,
    estimateSize: size,
    overscan: 6,
    rangeExtractor,
  });
  return (
    <ul
      aria-label={label}
      style={{
        height: virtualizer.getTotalSize(),
        position: "relative",
        flexShrink: 0,
        listStyle: "none",
        padding: 0,
        margin: 0,
      }}
    >
      {virtualizer.getVirtualItems().map((row) => (
        <li
          key={row.key}
          ref={virtualizer.measureElement}
          data-index={row.index}
          aria-posinset={row.index + 1}
          aria-setsize={items.length}
          onFocusCapture={() => setFocusedKey(getKey(items[row.index]))}
          style={{ position: "absolute", top: 0, left: 0, width: "100%", transform: `translateY(${row.start}px)` }}
        >
          {renderItem(items[row.index])}
        </li>
      ))}
    </ul>
  );
}
