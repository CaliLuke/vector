/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { act, type ReactNode } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, expect, it, vi } from "vitest";
import { Dropdown } from "./single-select";
import { MultiSelectDropdown } from "./multi-select";
import type { TDropdownOption } from "./dropdown";

const roots: Root[] = [];
const containers: HTMLDivElement[] = [];

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

if (typeof window.PointerEvent === "undefined") {
  class TestPointerEvent extends MouseEvent {
    readonly pointerType: string;

    constructor(type: string, init: PointerEventInit = {}) {
      super(type, init);
      this.pointerType = init.pointerType ?? "mouse";
    }
  }
  Object.assign(window, { PointerEvent: TestPointerEvent });
}

if (typeof window.ResizeObserver === "undefined") {
  class TestResizeObserver implements ResizeObserver {
    private readonly targets = new Set<Element>();

    constructor(private readonly callback: ResizeObserverCallback) {}

    observe(target: Element) {
      this.targets.add(target);
      this.notify();
    }

    unobserve(target: Element) {
      this.targets.delete(target);
    }

    disconnect() {
      this.targets.clear();
    }

    private notify() {
      const entries = [...this.targets].map((target): ResizeObserverEntry => ({
        target,
        contentRect: target.getBoundingClientRect(),
        borderBoxSize: [],
        contentBoxSize: [],
        devicePixelContentBoxSize: [],
      }));
      this.callback(entries, this);
    }
  }
  Object.assign(window, { ResizeObserver: TestResizeObserver });
  Object.assign(globalThis, { ResizeObserver: TestResizeObserver });
}

afterEach(async () => {
  await act(async () => {
    for (const root of roots.splice(0)) root.unmount();
  });
  for (const container of containers.splice(0)) container.remove();
});

function option(value: string, label: string, search = label, disabled = false): TDropdownOption {
  return { value, data: { [value]: value, label, search }, disabled };
}

function mountDropdown(element: ReactNode) {
  const container = document.createElement("div");
  document.body.append(container);
  const root = createRoot(container);
  roots.push(root);
  containers.push(container);
  act(() => root.render(element));
  return container;
}

async function click(element: Element) {
  await act(async () => {
    element.dispatchEvent(
      new window.PointerEvent("pointerdown", { button: 0, pointerType: "mouse", bubbles: true, cancelable: true })
    );
    const allowFocus = element.dispatchEvent(
      new MouseEvent("mousedown", { button: 0, bubbles: true, cancelable: true })
    );
    if (allowFocus && element instanceof HTMLElement) element.focus();
    element.dispatchEvent(
      new window.PointerEvent("pointerup", { button: 0, pointerType: "mouse", bubbles: true, cancelable: true })
    );
    element.dispatchEvent(new MouseEvent("mouseup", { button: 0, bubbles: true, cancelable: true }));
    element.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true }));
  });
}

async function press(element: Element, key: string) {
  await act(async () => {
    element.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true }));
  });
}

async function enterText(input: HTMLInputElement, value: string) {
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set?.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}

function renderedLabels(container: Element) {
  return [...container.querySelectorAll<HTMLElement>('[role="option"]')].map((item) => item.textContent?.trim());
}

function renderItem({ value, selected, disabled }: { value: string; selected: boolean; disabled?: boolean }) {
  return <span>{`${value}${selected ? " selected" : ""}${disabled ? " disabled" : ""}`}</span>;
}

it("single-select searches the supplied fields, selects an option, and closes", async () => {
  const onChange = vi.fn();
  const onOpen = vi.fn();
  const onClose = vi.fn();
  const options = [option("alpha", "Alpha", "first entry"), option("beta", "Beta", "second entry")];
  const container = mountDropdown(
    <Dropdown
      value=""
      onChange={onChange}
      onOpen={onOpen}
      onClose={onClose}
      options={options}
      keyExtractor={(item) => item.value}
      queryArray={["search", "label"]}
      inputPlaceholder="Find an option"
      renderItem={renderItem}
      buttonContent={(isOpen) => (isOpen ? "Close choices" : "Choose")}
    />
  );

  await click(container.querySelector("button")!);
  expect(onOpen).toHaveBeenCalledOnce();
  expect(container.querySelector('[role="listbox"]')).not.toBeNull();
  const input = container.querySelector<HTMLInputElement>('input[placeholder="Find an option"]')!;
  expect(document.activeElement).toBe(input);
  expect(renderedLabels(container)).toEqual(["alpha", "beta"]);

  await enterText(input, "second");
  expect(renderedLabels(container)).toEqual(["beta"]);

  await click(container.querySelector<HTMLElement>('[role="option"]')!);
  expect(onChange).toHaveBeenLastCalledWith("beta");
  expect(container.querySelector('[role="listbox"]')).toBeNull();
  expect(onClose).toHaveBeenCalledOnce();
});

it("single-select prioritizes the preferred option, sorts the rest by label, and skips disabled options", async () => {
  const onChange = vi.fn();
  const options = [
    option("last", "Last"),
    option("preferred", "Preferred"),
    option("blocked", "Blocked", "Blocked", true),
  ];
  const container = mountDropdown(
    <Dropdown
      value=""
      onChange={onChange}
      options={options}
      keyExtractor={(item) => item.value}
      firstItem={(value) => value === "preferred"}
      sortByKey="label"
      renderItem={renderItem}
      buttonContent={() => "Choose"}
    />
  );
  const trigger = container.querySelector<HTMLButtonElement>("button")!;
  expect(trigger.disabled).toBe(false);
  await click(trigger);

  expect(renderedLabels(container)).toEqual(["preferred", "blocked disabled", "last"]);
  const blocked = container.querySelector<HTMLElement>('[role="option"][aria-disabled="true"]')!;
  await click(blocked);
  expect(onChange).not.toHaveBeenCalled();
  expect(container.querySelector('[role="listbox"]')).not.toBeNull();

  const disabledContainer = mountDropdown(
    <Dropdown
      value=""
      onChange={onChange}
      options={options}
      keyExtractor={(item) => item.value}
      disabled
      buttonContent={() => "Disabled"}
    />
  );
  const disabledTrigger = disabledContainer.querySelector<HTMLButtonElement>("button")!;
  expect(disabledTrigger.disabled).toBe(true);
  await click(disabledTrigger);
  expect(disabledContainer.querySelector('[role="listbox"]')).toBeNull();
});

it("single-select applies firstItem without a sort key and compares selected option values exactly", async () => {
  const container = mountDropdown(
    <Dropdown
      value="item-10"
      onChange={vi.fn()}
      options={[option("item-1", "Item 1"), option("item-10", "Item 10")]}
      keyExtractor={(item) => item.value}
      firstItem={() => false}
      renderItem={renderItem}
      buttonContent={() => "Choose"}
    />
  );

  await click(container.querySelector("button")!);
  expect(renderedLabels(container)).toEqual(["item-10 selected", "item-1"]);
});

it("sorts numeric option data without converting it to text", async () => {
  const options: TDropdownOption[] = [
    { value: "ten", data: { ten: "ten", rank: 10 } },
    { value: "two", data: { two: "two", rank: 2 } },
  ];
  const container = mountDropdown(
    <Dropdown
      value=""
      onChange={vi.fn()}
      options={options}
      keyExtractor={(item) => item.value}
      sortByKey="rank"
      renderItem={renderItem}
      buttonContent={() => "Choose"}
    />
  );

  await click(container.querySelector("button")!);
  expect(renderedLabels(container)).toEqual(["two", "ten"]);
});

it("closes a single-select on Escape or an outside click, while preserving prevented outside clicks", async () => {
  const onClose = vi.fn();
  const container = mountDropdown(
    <Dropdown
      value=""
      onChange={vi.fn()}
      onClose={onClose}
      options={[option("one", "One")]}
      keyExtractor={(item) => item.value}
      disableSearch
      buttonContent={() => "Choose"}
    />
  );
  await click(container.querySelector("button")!);
  const protectedArea = document.createElement("div");
  protectedArea.dataset.preventOutsideClick = "";
  document.body.append(protectedArea);
  await act(async () => {
    protectedArea.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
  });
  expect(container.querySelector('[role="listbox"]')).not.toBeNull();

  await press(container.querySelector('[role="listbox"]')!, "Escape");
  expect(container.querySelector('[role="listbox"]')).toBeNull();
  expect(onClose).toHaveBeenCalledOnce();

  await click(container.querySelector("button")!);
  await act(async () => {
    document.body.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
  });
  expect(container.querySelector('[role="listbox"]')).toBeNull();
  expect(onClose).toHaveBeenCalledTimes(2);
  protectedArea.remove();
});

it("multi-select prioritizes firstItem and selected values, stays open after selection, and closes on Escape", async () => {
  const onChange = vi.fn();
  const onClose = vi.fn();
  const options = [option("other", "Other"), option("selected", "Selected"), option("preferred", "Preferred")];
  const container = mountDropdown(
    <MultiSelectDropdown
      value={["selected"]}
      onChange={onChange}
      onClose={onClose}
      options={options}
      keyExtractor={(item) => item.value}
      firstItem={(value) => value === "preferred"}
      sortByKey="label"
      renderItem={renderItem}
      buttonContent={(isOpen) => (isOpen ? "Open" : "Choose multiple")}
    />
  );

  await click(container.querySelector("button")!);
  expect(renderedLabels(container)).toEqual(["preferred", "selected selected", "other"]);
  await click(container.querySelector<HTMLElement>('[role="option"]')!);
  expect(onChange).toHaveBeenLastCalledWith(["selected", "preferred"]);
  expect(container.querySelector('[role="listbox"]')).not.toBeNull();

  await press(container.querySelector('[role="listbox"]')!, "Escape");
  expect(container.querySelector('[role="listbox"]')).toBeNull();
  expect(onClose).toHaveBeenCalledOnce();
});

it("renders loading and empty states, honors disableSorting, and uses real Popper positioning", async () => {
  const loadingContainer = mountDropdown(
    <Dropdown
      value=""
      onChange={vi.fn()}
      options={undefined}
      keyExtractor={(item) => item.value}
      loader={<span>Loading options</span>}
      disableSearch
      placement="top-start"
      containerClassName={(isOpen) => (isOpen ? "is-open" : "is-closed")}
      buttonContent={() => "Load"}
    />
  );
  const trigger = loadingContainer.querySelector<HTMLButtonElement>("button")!;
  trigger.getBoundingClientRect = () => new DOMRect(120, 80, 100, 30);
  await click(trigger);
  const panel = loadingContainer.querySelector<HTMLElement>('[role="listbox"]')!;
  expect(panel.textContent).toContain("Loading options");
  expect(panel.getAttribute("data-popper-placement")).toBe("top-start");
  expect(panel.style.position).toBe("absolute");
  expect(panel.style.transform).not.toBe("translate(0px, 0px)");
  expect(loadingContainer.firstElementChild?.className).toContain("is-open");

  const emptyContainer = mountDropdown(
    <Dropdown
      value=""
      onChange={vi.fn()}
      options={[]}
      keyExtractor={(item) => item.value}
      disableSearch
      disableSorting
      buttonContent={() => "Empty"}
    />
  );
  await click(emptyContainer.querySelector("button")!);
  expect(emptyContainer.textContent).toContain("No matching results");
});

it("clears a search query on Escape before closing and restores the empty single value", async () => {
  const onChange = vi.fn();
  const onClose = vi.fn();
  const container = mountDropdown(
    <Dropdown
      value=""
      onChange={onChange}
      onClose={onClose}
      options={[option("one", "One")]}
      keyExtractor={(item) => item.value}
      queryArray={["label"]}
      buttonContent={() => "Choose"}
    />
  );
  await click(container.querySelector("button")!);
  const input = container.querySelector<HTMLInputElement>("input")!;
  await enterText(input, "missing");
  await press(input, "Escape");
  expect(input.value).toBe("");
  expect(container.querySelector('[role="listbox"]')).not.toBeNull();
  await press(input, "Escape");
  expect(container.querySelector('[role="listbox"]')).toBeNull();
  expect(onClose).toHaveBeenCalledOnce();

  await click(container.querySelector("button")!);
  await click(container.querySelector<HTMLElement>('[role="option"]')!);
  expect(onChange).toHaveBeenLastCalledWith("one");
});
