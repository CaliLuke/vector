import assert from "node:assert/strict";
import test from "node:test";
import { setup, act } from "./helpers/issue-lifecycle-fixture.mjs";
import { clickWithPointer } from "../../../tests/frontend/helpers/pointer.mjs";

await test("detached peek keeps its selected layout in the portal and dismissal preserves the source row focus", async (t) => {
  const view = await setup(t, "peek", { embedIssue: false, portal: true });
  const browserErrors = [];
  const browserWindow = window;
  const onError = (event) => browserErrors.push(event.error);
  browserWindow.addEventListener("error", onError);
  t.after(() => browserWindow.removeEventListener("error", onError));
  const source = document.createElement("button");
  source.id = "issue-item";
  source.textContent = "Original work item";
  document.body.append(source);
  t.after(() => source.remove());
  assert.equal(Boolean(view.environment.container.querySelector('[data-action="delete"]')), false);
  const layoutButton = () => view.container.querySelector('button[aria-haspopup="listbox"]');
  assert.ok(layoutButton(), "detached peek exposes its layout picker");
  const propertiesShareDetailsColumn = () =>
    view.container
      .querySelector("[data-peek-properties]")
      .parentElement.contains(view.container.querySelector("[data-peek-details]"));
  assert.equal(propertiesShareDetailsColumn(), true);
  await clickWithPointer(layoutButton());
  const options = () => [...view.container.ownerDocument.querySelectorAll('[role="option"]')];
  assert.equal(options().length, 3);
  await clickWithPointer(options().find((option) => option.textContent.includes("common.full_screen")));
  assert.equal(propertiesShareDetailsColumn(), false, "full-screen layout places properties in their own column");
  assert.equal(view.state.closed, 0);
  await clickWithPointer(layoutButton());
  await clickWithPointer(options().find((option) => option.textContent.includes("common.modal")));
  assert.equal(propertiesShareDetailsColumn(), true, "modal layout reunites details and properties");
  await act(async () => {
    document.activeElement?.blur();
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));
  });
  assert.equal(view.state.closed, 1);
  assert.equal(document.activeElement, source);
  await view.render();
  assert.equal(Boolean(view.container.querySelector('[data-action="delete"]')), false);
  assert.deepEqual(browserErrors, []);
});

await test("embedded peek hides layout switching and closes only through its explicit notification dismissal", async (t) => {
  let dismissed = 0;
  const view = await setup(t, "peek", {
    onDismiss: () => {
      dismissed++;
    },
  });
  assert.equal(Boolean(view.container.querySelector('button[aria-haspopup="listbox"]')), false);
  await act(async () => document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true })));
  assert.equal(view.state.closed, 0);
  assert.equal(dismissed, 0);
  await view.click(view.container.querySelector("button"));
  assert.equal(view.state.closed, 1);
  assert.equal(dismissed, 1);
});

await test("opening a detached peek in its full page removes the selected peek without firing notification dismissal", async (t) => {
  let dismissed = 0;
  const view = await setup(t, "peek", {
    embedIssue: false,
    onDismiss: () => {
      dismissed++;
    },
  });
  const link = view.container.querySelector("a");
  assert.equal(link?.getAttribute("href"), "/acme/browse/P1-1/");
  await view.click(link);
  assert.equal(view.state.closed, 1);
  assert.equal(dismissed, 0);
});
