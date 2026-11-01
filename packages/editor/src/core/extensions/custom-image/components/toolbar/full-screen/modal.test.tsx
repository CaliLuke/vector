import { act, useState } from "react";
import { createRoot } from "react-dom/client";
import { expect, test, vi } from "vitest";
import { ImageFullScreenModal } from "./modal";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

function Viewer() {
  const [open, setOpen] = useState(false);
  return (
    <>
      <button type="button" onClick={() => setOpen(true)}>
        View attachment
      </button>
      <ImageFullScreenModal
        aspectRatio={2}
        downloadSrc="https://example.com/download"
        src="https://example.com/image"
        width="800px"
        isFullScreenEnabled={open}
        isTouchDevice={false}
        toggleFullScreenMode={setOpen}
      />
    </>
  );
}

test("fullscreen images own modal focus, support keyboard zoom and panning, and restore the trigger", async () => {
  const host = document.createElement("div");
  document.body.append(host);
  const root = createRoot(host);

  try {
    await act(async () => root.render(<Viewer />));
    const trigger = host.querySelector("button");
    if (!trigger) throw new Error("Missing viewer trigger");
    await act(async () => {
      trigger.focus();
      trigger.click();
    });
    const dialog = document.querySelector('[role="dialog"]');
    expect(dialog?.getAttribute("aria-label")).toBe("Fullscreen image viewer");
    await vi.waitFor(() => expect(dialog?.contains(document.activeElement)).toBe(true));
    const pan = dialog?.querySelector<HTMLButtonElement>('button[aria-label^="Pan image"]');
    const image = dialog?.querySelector("img");
    if (!pan || !image) throw new Error("Missing image controls");
    await act(async () => {
      pan.focus();
      pan.dispatchEvent(new KeyboardEvent("keydown", { key: "+", bubbles: true }));
    });
    expect(image.style.transform).toBe("scale(1.5)");
    await act(async () => pan.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowRight", bubbles: true })));
    expect(parseFloat(image.style.left)).toBeCloseTo(40 / 1.5);
    await act(async () => pan.click());
    expect(image.style.left).toBe("0px");
    await act(async () => pan.dispatchEvent(new KeyboardEvent("keydown", { key: "-", bubbles: true })));
    expect(image.style.transform).toBe("scale(1)");
    await act(async () => pan.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true })));
    await vi.waitFor(() => expect(document.querySelector('[role="dialog"]')).toBeNull());
    await vi.waitFor(() => expect(document.activeElement).toBe(trigger));
    expect(document.getElementById("editor-portal")).toBeNull();
  } finally {
    await act(async () => root.unmount());
    host.remove();
  }
});
