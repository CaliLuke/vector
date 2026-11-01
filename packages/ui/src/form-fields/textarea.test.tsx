import { act, createRef } from "react";
import { createRoot } from "react-dom/client";
import { expect, test, vi } from "vitest";
import { TextArea } from "./textarea";

test("textarea forwards object and callback refs while resizing the same element", () => {
  const container = document.createElement("div");
  const root = createRoot(container);
  const ref = createRef<HTMLTextAreaElement>();
  const cleanup = vi.fn();
  const callback = vi.fn(() => cleanup);
  try {
    act(() => root.render(<TextArea ref={ref} value="first" readOnly />));
    const element = container.querySelector("textarea");
    expect(ref.current).toBe(element);
    if (!element) throw new Error("Missing textarea");
    Object.defineProperty(element, "scrollHeight", { configurable: true, value: 96 });
    act(() => root.render(<TextArea ref={ref} value="second" readOnly />));
    expect(element.style.height).toBe("96px");
    Object.defineProperty(element, "scrollHeight", { configurable: true, value: 128 });
    act(() => root.render(<TextArea ref={ref} value="second" style={{ fontSize: 24 }} readOnly />));
    expect(element.style.height).toBe("128px");
    act(() => root.render(<TextArea ref={callback} value="second" readOnly />));
    expect(ref.current).toBeNull();
    expect(callback).toHaveBeenCalledExactlyOnceWith(element);
    act(() => root.render(null));
    expect(cleanup).toHaveBeenCalledTimes(1);
  } finally {
    act(() => root.unmount());
  }
});
