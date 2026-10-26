import { attachInstruction, extractInstruction } from "@atlaskit/pragmatic-drag-and-drop-hitbox/tree-item";
import type {
  dropTargetForElements,
  ElementDragPayload,
} from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";

export function treeDropPosition(data: Parameters<typeof extractInstruction>[0], isLastChild: boolean) {
  const instruction = extractInstruction(data)?.type;
  if (!instruction) return undefined;
  return instruction === "reorder-below" && isLastChild ? "DRAG_BELOW" : "DRAG_OVER";
}

/** Native hitbox and drag feedback for flat Gantt and navigation rows. */
export function treeDropPositionTarget(
  id: string,
  isLastChild: boolean,
  setPosition: (position: ReturnType<typeof treeDropPosition>) => void,
  drop?: (
    sourceId: string | undefined,
    targetId: string | undefined,
    below: boolean,
    element: ElementDragPayload["element"]
  ) => void
): Pick<Parameters<typeof dropTargetForElements>[0], "getData" | "onDrag" | "onDragLeave" | "onDrop"> {
  return {
    getData: ({ input, element }) =>
      attachInstruction(
        { id },
        {
          input,
          element,
          currentLevel: 0,
          indentPerLevel: 0,
          mode: isLastChild ? "last-in-group" : "standard",
        }
      ),
    onDrag: ({ self }) => setPosition(treeDropPosition(self.data, isLastChild)),
    onDragLeave: () => setPosition(undefined),
    onDrop: ({ self, source }) => {
      setPosition(undefined);
      const position = treeDropPosition(self.data, isLastChild);
      if (!position) return;
      drop?.(
        typeof source.data.id === "string" ? source.data.id : undefined,
        typeof self.data.id === "string" ? self.data.id : undefined,
        position === "DRAG_BELOW",
        source.element
      );
    },
  };
}
