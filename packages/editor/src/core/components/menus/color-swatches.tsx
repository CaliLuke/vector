/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Ban } from "lucide-react";
import { cn } from "@plane/utils/common";
import { COLORS_LIST } from "@/constants/common";

type PaletteColor = (typeof COLORS_LIST)[number];

export function ColorSwatches({
  tone,
  onSelect,
  resetClassName = "hover:bg-layer-1",
}: {
  tone: "text" | "background";
  onSelect: (color: PaletteColor | undefined) => void;
  resetClassName?: string;
}) {
  return (
    <>
      {COLORS_LIST.map((color) => (
        <button
          key={color.key}
          aria-label={color.label}
          type="button"
          className="size-6 flex-shrink-0 rounded-sm border-[0.5px] border-strong-1 transition-opacity hover:opacity-60"
          style={{ backgroundColor: tone === "text" ? color.textColor : color.backgroundColor }}
          onClick={() => onSelect(color)}
        />
      ))}
      <button
        type="button"
        aria-label={tone === "text" ? "Reset text color" : "Reset background color"}
        className={cn(
          "grid size-6 flex-shrink-0 place-items-center rounded-sm border-[0.5px] border-strong-1 text-tertiary transition-colors",
          resetClassName
        )}
        onClick={() => onSelect(undefined)}
      >
        <Ban className="size-4" />
      </button>
    </>
  );
}
