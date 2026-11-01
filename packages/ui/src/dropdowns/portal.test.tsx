import { act } from "react";
import { createRoot, hydrateRoot } from "react-dom/client";
import { renderToString } from "react-dom/server";
import { expect, test } from "vitest";
import { DropdownPortal } from "./portal";

test("dropdown portals hydrate without server markup and release their contents", async () => {
  const container = document.createElement("div");
  const target = document.createElement("div");
  document.body.append(container, target);
  const content = (
    <DropdownPortal container={target} wrap>
      <button>Hydrated action</button>
    </DropdownPortal>
  );
  container.innerHTML = renderToString(content);
  expect(container.innerHTML).toBe("");
  const root = hydrateRoot(container, content);
  try {
    await act(async () => {});
    expect(target.querySelector("[data-radix-portal] button")?.textContent).toBe("Hydrated action");
  } finally {
    act(() => root.unmount());
    expect(target.innerHTML).toBe("");
    container.remove();
    target.remove();
  }
});

test("dropdown portals use the document body when no target is supplied", () => {
  const container = document.createElement("div");
  const root = createRoot(container);
  try {
    act(() =>
      root.render(
        <DropdownPortal>
          <button data-test-portal-action>Action</button>
        </DropdownPortal>
      )
    );
    expect(document.querySelector("[data-test-portal-action]")?.parentElement).toBe(document.body);
  } finally {
    act(() => root.unmount());
    expect(document.querySelector("[data-test-portal-action]")).toBeNull();
  }
});
