import { React } from "./react.mjs";

// Browser button activation includes pointer/mouse events before click. Headless UI
// opens its combobox on mousedown, which HTMLElement.click() alone does not emit.
export async function clickWithPointer(element) {
  await React.act(async () => {
    element.dispatchEvent(
      new window.PointerEvent("pointerdown", { button: 0, pointerType: "mouse", bubbles: true, cancelable: true })
    );
    const allowFocus = element.dispatchEvent(
      new MouseEvent("mousedown", { button: 0, bubbles: true, cancelable: true })
    );
    if (allowFocus) element.focus();
    element.dispatchEvent(
      new window.PointerEvent("pointerup", { button: 0, pointerType: "mouse", bubbles: true, cancelable: true })
    );
    element.dispatchEvent(new MouseEvent("mouseup", { button: 0, bubbles: true, cancelable: true }));
  });
  await React.act(async () => element.click());
}
