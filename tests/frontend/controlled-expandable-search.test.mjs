import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { pathToFileURL } from "node:url";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, webRequire } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const stubs = {
  "@plane/propel/icons": `import React from "${pathToFileURL(webRequire.resolve("react")).href}";
    export const SearchIcon = (props) => React.createElement("svg", props);
    export const CloseIcon = (props) => React.createElement("svg", props);`,
  "@plane/propel/icon-button": `import React from "${pathToFileURL(webRequire.resolve("react")).href}";
    export const IconButton = ({ icon: Icon, className, ...props }) =>
      React.createElement("button", { ...props, className }, React.createElement(Icon));`,
  "@plane/utils/common":
    'export const cn = (...parts) => parts.flatMap(part =>\n    typeof part === "string" ? [part] : Object.entries(part ?? {}).filter(([, enabled]) => enabled).map(([name]) => name)\n  ).join(" ");',
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@plane/hooks") {
      return nextResolve(
        new URL("../../packages/hooks/src/use-outside-click-detector.tsx", import.meta.url).href,
        context
      );
    }
    if (Object.hasOwn(stubs, specifier)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(stubs[specifier])}` };
    }
    return nextResolve(specifier, context);
  },
});

const EMPTY_SEARCH_PROPS = {};

function SearchHarness({ initialValue = "", searchProps = EMPTY_SEARCH_PROPS } = EMPTY_SEARCH_PROPS) {
  const [value, setValue] = React.useState(initialValue);
  return React.createElement("div", null, [
    React.createElement(
      "button",
      { key: "set-query", "data-testid": "set-query", onClick: () => setValue("Cycle planning") },
      "Set query externally"
    ),
    React.createElement("button", { key: "outside", "data-testid": "outside" }, "Outside"),
    React.createElement(SearchControl, {
      key: "search",
      ...searchProps,
      value,
      onChange: setValue,
      placeholder: "Search projects",
    }),
  ]);
}

const { ControlledExpandableSearch: SearchControl } =
  await import("../../apps/web/core/components/common/controlled-expandable-search.tsx");
const { toggleCatalogueFilterValues } =
  await import("../../apps/web/core/components/cycles-modules/toggle-catalogue-filter-values.ts");

function changeInput(input, value) {
  Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set.call(input, value);
  input.dispatchEvent(new Event("input", { bubbles: true }));
}

await test("expandable search mounts and focuses its input only after opening", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  await env.render(React.createElement(SearchHarness));

  assert.equal(env.container.querySelector("input"), null, "collapsed search has no hidden tab stop");
  const trigger = env.container.querySelector('button[aria-label="Search projects"]');
  assert.ok(trigger);
  await React.act(async () => trigger.click());

  const input = env.container.querySelector('input[aria-label="Search projects"]');
  assert.ok(input);
  assert.equal(document.activeElement, input, "opening search transfers focus to the mounted input");
  assert.equal(trigger.getAttribute("aria-expanded"), "true");
});

await test("Escape clears a query before closing and returns focus to the search trigger", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  await env.render(React.createElement(SearchHarness));
  const trigger = env.container.querySelector('button[aria-label="Search projects"]');
  await React.act(async () => trigger.click());
  const input = env.container.querySelector("input");

  await React.act(async () => changeInput(input, "Planning"));
  await React.act(async () =>
    input.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }))
  );
  assert.equal(env.container.querySelector("input").value, "", "first Escape clears the current query");

  await React.act(async () =>
    env.container
      .querySelector("input")
      .dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }))
  );
  assert.equal(env.container.querySelector("input"), null, "second Escape closes an empty search");
  assert.equal(document.activeElement, trigger, "closing search returns focus to its trigger");
});

await test("outside click closes only an empty search", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  await env.render(React.createElement(SearchHarness));
  const trigger = env.container.querySelector('button[aria-label="Search projects"]');
  const outside = env.container.querySelector('[data-testid="outside"]');
  await React.act(async () => trigger.click());

  await React.act(async () => outside.dispatchEvent(new MouseEvent("mousedown", { bubbles: true })));
  assert.equal(env.container.querySelector("input"), null, "empty search collapses after outside click");

  await React.act(async () => trigger.click());
  const input = env.container.querySelector("input");
  await React.act(async () => changeInput(input, "Planning"));
  await React.act(async () => outside.dispatchEvent(new MouseEvent("mousedown", { bubbles: true })));
  assert.equal(env.container.querySelector("input").value, "Planning", "nonempty search stays open outside");
});

await test("catalogue search opens for a saved query without stealing initial focus", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  await env.render(
    React.createElement(SearchHarness, {
      initialValue: "Cycle planning",
      searchProps: { initiallyOpen: true, openWhenValueExists: true },
    })
  );

  const input = env.container.querySelector('input[aria-label="Search projects"]');
  assert.ok(input);
  assert.equal(input.value, "Cycle planning");
  assert.notEqual(document.activeElement, input, "saved search does not steal focus when the header mounts");
});

await test("cycle-style search opens when another control updates the query", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  await env.render(React.createElement(SearchHarness, { searchProps: { openWhenValueExists: true } }));
  assert.equal(env.container.querySelector("input"), null);

  await React.act(async () => env.container.querySelector('[data-testid="set-query"]').click());
  assert.equal(env.container.querySelector("input").value, "Cycle planning");
});

await test("module-style close preserves the query and can be reopened", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  await env.render(
    React.createElement(SearchHarness, {
      initialValue: "Module planning",
      searchProps: { initiallyOpen: true, clearOnClose: false },
    })
  );

  const input = env.container.querySelector("input");
  const close = env.container.querySelector('button[aria-label="Close Search projects"]');
  assert.ok(input);
  assert.ok(close);
  await React.act(async () => close.click());
  assert.equal(env.container.querySelector("input"), null);

  const trigger = env.container.querySelector('button[aria-label="Search projects"]');
  assert.ok(trigger);
  await React.act(async () => trigger.click());
  assert.equal(env.container.querySelector("input").value, "Module planning");
});

await test("catalogue filter toggles preserve order without mutating the preference snapshot", () => {
  const current = ["active", "completed"];
  const next = toggleCatalogueFilterValues(current, ["completed", "upcoming"]);

  assert.deepEqual(next, ["active", "upcoming"]);
  assert.deepEqual(current, ["active", "completed"]);
  assert.deepEqual(toggleCatalogueFilterValues(undefined, "active"), ["active"]);
});
