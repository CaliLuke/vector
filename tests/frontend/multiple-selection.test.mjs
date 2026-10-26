import assert from "node:assert/strict";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { createQueryRouterFixture } from "./helpers/query-router.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { MultipleSelectGroup } = await import("../../apps/web/core/components/core/multiple-select/select-group.tsx");
const { IssueBulkOperationsRoot } = await import("../../apps/web/core/components/issues/bulk-operations/root.tsx");
const { advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
const h = React.createElement;
const defaultEntities = { first: ["a", "b", "c"], second: ["d"] };

function SignalOutput() {
  const [value, setValue] = React.useState("before");
  return h(
    React.Fragment,
    null,
    h("output", { "data-signal": true }, value),
    h("button", { "data-signal-update": true, onClick: () => setValue("after") }, "Update callback")
  );
}

function Selection({ name = "selection", entities = defaultEntities, disabled = false, signal }) {
  const ref = React.useRef(null);
  return h(
    "section",
    { ref, "data-selection": name },
    h(MultipleSelectGroup, { containerRef: ref, entities, disabled }, (helpers) =>
      h(
        React.Fragment,
        null,
        signal && h(SignalOutput),
        h("input", { "aria-label": "draft" }),
        Object.entries(entities).map(([group, ids]) =>
          h(
            React.Fragment,
            { key: group },
            h(
              "button",
              { key: group, "data-group": group, onClick: () => helpers.handleGroupClick(group) },
              helpers.isGroupSelected(group)
            ),
            ids.map((id) =>
              h(
                "button",
                {
                  key: group + id,
                  "data-item": id,
                  "data-entity-id": id,
                  "data-entity-group-id": group,
                  "aria-pressed": helpers.getIsEntitySelected(id),
                  "data-active": helpers.getIsEntityActive(id),
                  onClick: (event) => helpers.handleEntityClick(event, id, group),
                },
                id
              )
            )
          )
        ),
        h(IssueBulkOperationsRoot, { selectionHelpers: helpers })
      )
    )
  );
}

async function setup(t, props) {
  const fixture = await createQueryRouterFixture(t);
  await fixture.render(h(Selection, props));
  const container = fixture.environment.container;
  const selected = () =>
    [...container.querySelectorAll("[data-item][aria-pressed=true]")].map((node) => node.dataset.item);
  const click = async (selector, options = {}) => {
    await act(async () =>
      container.querySelector(selector).dispatchEvent(new MouseEvent("click", { bubbles: true, ...options }))
    );
  };
  return { ...fixture, container, selected, click };
}

void test("local selection supports toggles, shift ranges, group selection and clear", async (t) => {
  const fixture = await setup(t);
  await fixture.click('[data-item="a"]');
  await fixture.click('[data-item="c"]', { shiftKey: true });
  assert.deepEqual(fixture.selected(), ["a", "b", "c"]);
  assert.match(fixture.container.textContent, /Clear selection/);

  await fixture.click('[data-group="first"]');
  assert.deepEqual(fixture.selected(), []);
  await fixture.click('[data-group="second"]');
  assert.deepEqual(fixture.selected(), ["d"]);
  const clear = [...fixture.container.querySelectorAll("button")].find(
    (button) => button.textContent === "Clear selection"
  );
  await act(async () => clear.click());
  assert.deepEqual(fixture.selected(), []);
  assert.doesNotMatch(fixture.container.textContent, /Clear selection/);
});

void test("shift-clicking the range anchor preserves its selection", async (t) => {
  const fixture = await setup(t);
  await fixture.click('[data-item="a"]');
  await fixture.click('[data-item="a"]', { shiftKey: true });
  assert.deepEqual(fixture.selected(), ["a"]);
});

void test("entity removal prunes selection and its shift anchor", async (t) => {
  const fixture = await setup(t);
  await fixture.click('[data-item="a"]');
  await fixture.click('[data-item="b"]');
  await fixture.render(h(Selection, { entities: { first: ["a", "c"] } }));
  assert.deepEqual(fixture.selected(), ["a"]);
  await fixture.click('[data-item="c"]', { shiftKey: true });
  assert.deepEqual(fixture.selected(), ["a", "c"]);
});

void test("keyboard selection follows the active row and ignores text entry", async (t) => {
  const fixture = await setup(t);
  fixture.container.querySelector("section").scrollBy = () => {};
  await fixture.click('[data-item="a"]');
  await act(async () =>
    fixture.container
      .querySelector('[data-item="a"]')
      .dispatchEvent(
        new KeyboardEvent("keydown", { key: "ArrowDown", shiftKey: true, bubbles: true, cancelable: true })
      )
  );
  assert.deepEqual(fixture.selected(), ["a", "b"]);
  assert.equal(fixture.container.querySelector('[data-active="true"]').dataset.item, "b");
  await act(async () =>
    fixture.container
      .querySelector("input")
      .dispatchEvent(
        new KeyboardEvent("keydown", { key: "ArrowDown", shiftKey: true, bubbles: true, cancelable: true })
      )
  );
  assert.deepEqual(fixture.selected(), ["a", "b"]);
});

void test("route changes and same-route session changes discard selection", async (t) => {
  const fixture = await setup(t);
  await fixture.click('[data-item="a"]');
  await act(async () => fixture.router.navigate({ to: "/acme/projects/p2/issues/" }));
  assert.deepEqual(fixture.selected(), []);
  await fixture.click('[data-item="b"]');
  await act(async () => advanceIdentitySession(fixture.client));
  assert.deepEqual(fixture.selected(), []);
  const beforeUnload = new Event("beforeunload", { cancelable: true });
  window.dispatchEvent(beforeUnload);
  assert.equal(beforeUnload.defaultPrevented, false);
});

void test("disabling selection discards its state before it can be re-enabled", async (t) => {
  const fixture = await setup(t);
  await fixture.click('[data-item="a"]');
  await fixture.render(h(Selection, { disabled: true }));
  assert.deepEqual(fixture.selected(), []);
  await fixture.click('[data-item="b"]');
  await fixture.render(h(Selection));
  assert.deepEqual(fixture.selected(), []);
});

void test("independent groups do not share selection and render callbacks remain reactive", async (t) => {
  const fixture = await setup(t);
  await fixture.render(
    h(React.Fragment, null, h(Selection, { name: "left", signal: true }), h(Selection, { name: "right" }))
  );
  await fixture.click('[data-selection="left"] [data-item="a"]');
  assert.equal(
    fixture.container.querySelector('[data-selection="right"] [data-item="a"]').getAttribute("aria-pressed"),
    "false"
  );
  await act(async () => fixture.container.querySelector("[data-signal-update]").click());
  assert.equal(fixture.container.querySelector("output").textContent, "after");
});
