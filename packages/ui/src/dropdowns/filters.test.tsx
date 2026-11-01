import { act } from "react";
import type { ComponentProps } from "react";
import { createRoot } from "react-dom/client";
import { expect, test, vi } from "vitest";
import { FiltersDropdown } from "./filters";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

async function fixture(props: Partial<ComponentProps<typeof FiltersDropdown>>) {
  const host = document.createElement("div");
  document.body.append(host);
  const root = createRoot(host);
  const render = (next: Partial<ComponentProps<typeof FiltersDropdown>>) =>
    act(async () =>
      root.render(
        <FiltersDropdown {...props} {...next}>
          <input aria-label="Filter value" />
        </FiltersDropdown>
      )
    );
  await render({});
  const trigger = host.querySelector("button");
  if (!trigger) throw new Error("Missing filter trigger");
  return {
    host,
    trigger,
    render,
    close: async () => {
      await act(async () => root.unmount());
      host.remove();
    },
  };
}

test("responsive filter labels share a trigger and applied filters retain normal popup focus", async () => {
  const f = await fixture({
    title: "Filters",
    responsive: true,
    icon: <svg aria-label="Filter icon" />,
    miniIcon: <span>F</span>,
    isFiltersApplied: true,
  });
  try {
    expect(f.host.querySelectorAll("button")).toHaveLength(1);
    expect(f.trigger.textContent).toBe("FiltersF");
    await act(async () => {
      f.trigger.focus();
      f.trigger.click();
    });
    await vi.waitFor(() => expect(document.activeElement?.getAttribute("aria-label")).toBe("Filter value"));
    await act(async () =>
      document.activeElement?.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }))
    );
    await vi.waitFor(() => expect(f.trigger.getAttribute("aria-expanded")).toBe("false"));
    await f.render({ responsive: true, miniIcon: undefined, icon: undefined, isFiltersApplied: false, disabled: true });
    expect(f.trigger.textContent).toBe("FiltersFilters");
    await act(async () => f.trigger.click());
    expect(f.trigger.getAttribute("aria-expanded")).toBe("false");
  } finally {
    await f.close();
  }
});

test("mobile custom triggers do not force focus into the filter input", async () => {
  const userAgent = vi.spyOn(window.navigator, "userAgent", "get").mockReturnValue("iPhone");
  const f = await fixture({ menuButton: <span>Choose filters</span>, tabIndex: 3 });
  try {
    expect(f.trigger.textContent).toBe("Choose filters");
    expect(f.trigger.tabIndex).toBe(3);
    await act(async () => {
      f.trigger.focus();
      f.trigger.click();
    });
    await vi.waitFor(() => expect(document.querySelector('[aria-label="Filter value"]')).not.toBeNull());
    expect(document.activeElement?.getAttribute("aria-label")).not.toBe("Filter value");
  } finally {
    await f.close();
    userAgent.mockRestore();
  }
});

test("default filter labels and disabled custom triggers remain usable native buttons", async () => {
  const f = await fixture({});
  try {
    expect(f.trigger.textContent).toBe("Dropdown");
    await f.render({ menuButton: <span>Unavailable</span>, disabled: true });
    const trigger = f.host.querySelector("button")!;
    expect(trigger.disabled).toBe(true);
    await act(async () => trigger.click());
    expect(trigger.getAttribute("aria-expanded")).toBe("false");
  } finally {
    await f.close();
  }
});
