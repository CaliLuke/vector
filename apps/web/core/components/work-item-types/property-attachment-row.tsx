import { useEffect, useRef, useState } from "react";
import type { ReactNode } from "react";
import { GripVertical } from "lucide-react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { draggable, dropTargetForElements } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";
import { attachClosestEdge, extractClosestEdge } from "@atlaskit/pragmatic-drag-and-drop-hitbox/closest-edge";
import type { Edge } from "@atlaskit/pragmatic-drag-and-drop-hitbox/closest-edge";

type Props = {
  id: string;
  name: string;
  typeId: string;
  readOnly: boolean;
  onMove: (sourceId: string, targetId: string, after: boolean) => void;
  children: ReactNode;
};

export function PropertyAttachmentRow({ id, name, typeId, readOnly, onMove, children }: Props) {
  const rowRef = useRef<HTMLDivElement>(null);
  const handleRef = useRef<HTMLButtonElement>(null);
  const [edge, setEdge] = useState<Edge | null>(null);
  useEffect(() => {
    const element = rowRef.current;
    const dragHandle = handleRef.current;
    if (!element || !dragHandle || readOnly) return undefined;
    return combine(
      draggable({
        element,
        dragHandle,
        getInitialData: () => ({ kind: "type-property-attachment", typeId, id }),
      }),
      dropTargetForElements({
        element,
        canDrop: ({ source }) =>
          source.data.kind === "type-property-attachment" && source.data.typeId === typeId && source.data.id !== id,
        getData: ({ input }) => attachClosestEdge({ id }, { element, input, allowedEdges: ["top", "bottom"] }),
        onDrag: ({ self }) => setEdge(extractClosestEdge(self.data)),
        onDragLeave: () => setEdge(null),
        onDrop: ({ source, self }) => {
          setEdge(null);
          if (typeof source.data.id === "string")
            onMove(source.data.id, id, extractClosestEdge(self.data) === "bottom");
        },
      })
    );
  }, [id, typeId, readOnly, onMove]);
  return (
    <div ref={rowRef} className="relative flex flex-wrap items-center gap-3 border-b border-subtle py-3 last:border-0">
      {!readOnly && edge && (
        <div
          className={`pointer-events-none absolute inset-x-0 border-t-2 border-accent-strong ${edge === "top" ? "top-0" : "bottom-0"}`}
        />
      )}
      <button
        ref={handleRef}
        type="button"
        tabIndex={-1}
        aria-label={`Drag ${name} to reorder`}
        title="Drag to reorder, or use the move buttons"
        disabled={readOnly}
        className="cursor-grab text-tertiary disabled:cursor-default disabled:opacity-25"
      >
        <GripVertical className="size-3.5" />
      </button>
      {children}
    </div>
  );
}

export function reorderAttachedProperties(ids: string[], sourceId: string, targetId: string, after: boolean): string[] {
  if (sourceId === targetId || !ids.includes(sourceId) || !ids.includes(targetId)) return ids;
  const reordered = ids.filter((id) => id !== sourceId);
  reordered.splice(reordered.indexOf(targetId) + (after ? 1 : 0), 0, sourceId);
  return reordered.every((id, index) => id === ids[index]) ? ids : reordered;
}
