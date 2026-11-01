import { act } from "react";
import { createRoot } from "react-dom/client";
import { expect, test, vi } from "vitest";
import { ColorSwatches } from "./color-swatches";

test.each([
  ["text", "var(--editor-colors-green-text)", "Reset text color"],
  ["background", "var(--editor-colors-green-background)", "Reset background color"],
] as const)("%s swatches display the palette and emit color selection and reset", (tone, background, resetLabel) => {
  const host = document.createElement("div");
  const root = createRoot(host);
  const onSelect = vi.fn();
  try {
    act(() => root.render(<ColorSwatches tone={tone} onSelect={onSelect} />));
    const green = host.querySelector<HTMLButtonElement>('button[aria-label="Green"]')!;
    expect(green.style.backgroundColor).toBe(background);
    act(() => green.click());
    expect(onSelect).toHaveBeenLastCalledWith({
      key: "green",
      label: "Green",
      textColor: "var(--editor-colors-green-text)",
      backgroundColor: "var(--editor-colors-green-background)",
    });
    act(() => host.querySelector<HTMLButtonElement>(`button[aria-label="${resetLabel}"]`)!.click());
    expect(onSelect).toHaveBeenLastCalledWith(undefined);
  } finally {
    act(() => root.unmount());
  }
});
