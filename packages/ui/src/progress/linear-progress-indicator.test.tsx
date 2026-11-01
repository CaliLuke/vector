import { act } from "react";
import { createRoot } from "react-dom/client";
import { expect, test } from "vitest";
import { LinearProgressIndicator } from "./linear-progress-indicator";

test("progress segments keep proportional widths and omit zero totals", () => {
  const container = document.createElement("div");
  const root = createRoot(container);
  const complete = { id: "complete", name: "Completed", value: 3, color: "green" };
  const pending = { id: "pending", name: "Pending", value: 1, color: "blue" };
  try {
    act(() => root.render(<LinearProgressIndicator noTooltip data={[complete, pending]} />));
    expect(Array.from(container.querySelectorAll<HTMLElement>("[style]"), (bar) => bar.style.width)).toEqual([
      "75%",
      "25%",
    ]);
    act(() => root.render(<LinearProgressIndicator noTooltip data={[{ ...complete, value: 0 }]} />));
    expect(container.querySelector("[style]")).toBeNull();
    act(() => root.render(<LinearProgressIndicator noTooltip data={[]} />));
    expect(container.querySelector("[style]")).toBeNull();
  } finally {
    act(() => root.unmount());
  }
});
