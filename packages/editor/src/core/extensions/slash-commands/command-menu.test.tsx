import { act, createRef } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import type { CommandListInstance } from "@/helpers/tippy";
import type { ISlashCommandItem } from "@/types";
import type { TSlashCommandSection } from "./command-items-list";
import { SlashCommandsMenu } from "./command-menu";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
let host: HTMLDivElement;
let root: Root;
beforeEach(() => {
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  Object.defineProperty(HTMLElement.prototype, "scrollIntoView", { value: vi.fn(), configurable: true });
});
afterEach(() => {
  act(() => root.unmount());
  host.remove();
  Reflect.deleteProperty(HTMLElement.prototype, "scrollIntoView");
});
const item = (title: string): ISlashCommandItem => ({
  commandKey: "text",
  key: title,
  title,
  description: title,
  searchTerms: [],
  icon: null,
  command: () => {},
});

test("navigation skips empty sections and a consumed editor event is not handled twice", () => {
  const command = vi.fn();
  const sections: TSlashCommandSection[] = [
    { key: "general", items: [item("First")] },
    { key: "text-colors", items: [] },
    { key: "background-colors", items: [item("Last")] },
  ];
  const ref = createRef<CommandListInstance>();
  act(() =>
    root.render(<SlashCommandsMenu ref={ref} items={sections} query="" command={command} onClose={() => {}} />)
  );
  const down = new KeyboardEvent("keydown", { key: "ArrowDown", cancelable: true, bubbles: true });
  act(() => {
    ref.current?.onKeyDown({ event: down });
    document.dispatchEvent(down);
  });
  act(() => {
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", cancelable: true }));
  });
  expect(command).toHaveBeenCalledExactlyOnceWith(sections[2].items[0]);
});

test("new results reset selection and unmount releases document keyboard handlers", async () => {
  const command = vi.fn();
  const sections: TSlashCommandSection[] = [{ key: "general", items: [item("First"), item("Second")] }];
  const render = (items: TSlashCommandSection[]) =>
    act(() => root.render(<SlashCommandsMenu items={items} query="" command={command} onClose={() => {}} />));

  render(sections);
  await press("ArrowDown");
  const replacement: TSlashCommandSection[] = [{ key: "general", items: [item("Replacement")] }];
  render(replacement);
  await press("Enter");
  expect(command).toHaveBeenCalledExactlyOnceWith(replacement[0].items[0]);
  act(() => root.render(null));
  await press("Enter");
  expect(command).toHaveBeenCalledTimes(1);
});

async function press(key: string) {
  await act(async () => {
    document.dispatchEvent(new KeyboardEvent("keydown", { key, cancelable: true }));
  });
}
