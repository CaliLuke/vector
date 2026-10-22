import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const packageStubs = {
  "@plane/hooks": "export const useOutsideClickDetector = () => {};",
  "@plane/propel/icons": `export const ChevronDownIcon = () => null;
    export const ChevronRightIcon = () => null;
    export const CheckIcon = () => null;
    export const SearchIcon = () => null;`,
  "@plane/propel/tooltip": `export const Tooltip = ({ children }) => children;`,
  "react-popper": "export const usePopper = () => ({ styles: { popper: {} }, attributes: { popper: {} } });",
};
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "../utils" && context.parentURL?.includes("/packages/ui/src/dropdowns/")) {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export const cn = (...values) => values.filter(value => typeof value === 'string').join(' ');")}`,
      };
    }
    if (Object.hasOwn(packageStubs, specifier)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(packageStubs[specifier])}` };
    }
    return nextResolve(specifier, context);
  },
});

let dropdowns;
async function loadDropdowns() {
  dropdowns ??= Promise.all([
    import("../../packages/ui/src/dropdowns/custom-menu.tsx"),
    import("../../packages/ui/src/dropdowns/custom-select.tsx"),
    import("../../packages/ui/src/dropdowns/custom-search-select.tsx"),
  ]).then(([menu, select, search]) => ({
    CustomMenu: menu.CustomMenu,
    CustomSelect: select.CustomSelect,
    CustomSearchSelect: search.CustomSearchSelect,
  }));
  return dropdowns;
}

function keyDown(element, key) {
  return element.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true }));
}

async function activateWithKeyboard(element, key) {
  let useNativeClick;
  await React.act(() => {
    useNativeClick = keyDown(element, key);
  });
  // jsdom does not produce the native click for an uncancelled Enter/Space key.
  if (useNativeClick) await React.act(() => element.click());
}

async function waitForFocus(element, message = "focus should return to the trigger") {
  const deadline = performance.now() + 2_000;
  await React.act(async () => {
    await new Promise((resolve) => {
      const checkFocus = () => {
        if (document.activeElement === element || performance.now() >= deadline) resolve();
        else setTimeout(checkFocus, 10);
      };
      checkFocus();
    });
  });
  assert.ok(document.activeElement === element, message);
}

async function renderDropdown(environment, kind, callbacks = {}) {
  const { CustomMenu, CustomSelect, CustomSearchSelect } = await loadDropdowns();
  const props = {
    customButton: React.createElement("button", null, "Trigger"),
    customButtonAsChild: true,
    ...callbacks,
  };
  const renderers = {
    menu: () => React.createElement(CustomMenu, props, React.createElement(CustomMenu.MenuItem, null, "Option")),
    select: () =>
      React.createElement(
        CustomSelect,
        { value: null, onChange() {}, ...props },
        React.createElement(CustomSelect.Option, { value: "one" }, "Option")
      ),
    search: () =>
      React.createElement(CustomSearchSelect, {
        value: null,
        onChange() {},
        options: [{ value: "one", content: "Option", query: "option" }],
        ...props,
      }),
  };
  await environment.render(renderers[kind]());
  return environment.container.querySelector("button");
}

for (const kind of ["menu", "select", "search"]) {
  test(`${kind} trigger toggles on repeated non-pointer activation`, async (t) => {
    const environment = await createReactDomTestEnvironment();
    t.after(() => environment.unmount());
    const trigger = await renderDropdown(environment, kind);

    await React.act(() => trigger.click());
    assert.equal(trigger.getAttribute("aria-expanded"), "true");
    assert.ok(document.querySelector("[role=menu], [role=listbox]"));

    await React.act(() => trigger.click());
    assert.equal(trigger.getAttribute("aria-expanded"), "false");
    assert.ok(!document.querySelector("[role=menu], [role=listbox]"), "a second activation should close the popup");

    await React.act(() => trigger.click());
    assert.equal(trigger.getAttribute("aria-expanded"), "true");
    assert.ok(document.querySelector("[role=menu], [role=listbox]"));
  });

  for (const key of ["Enter", " ", "ArrowDown", "ArrowUp"]) {
    test(`${kind} opens once with ${JSON.stringify(key)} and closes with Escape`, async (t) => {
      const environment = await createReactDomTestEnvironment();
      t.after(() => environment.unmount());
      const trigger = await renderDropdown(environment, kind);

      await React.act(() => trigger.focus());
      await activateWithKeyboard(trigger, key);
      assert.equal(trigger.getAttribute("aria-expanded"), "true");
      const options = document.querySelector("[role=menu], [role=listbox]");
      assert.ok(options);

      const escapeTarget = kind === "menu" ? options : kind === "search" ? options.querySelector("input") : trigger;
      await waitForFocus(escapeTarget, "keyboard opening should focus the active menu or search control");
      await React.act(() => keyDown(escapeTarget, "Escape"));
      assert.equal(trigger.getAttribute("aria-expanded"), "false");
      assert.ok(!document.querySelector("[role=menu], [role=listbox]"), "Escape should close the popup");
      await waitForFocus(trigger);
    });
  }
}

test("CustomMenu composes a button trigger and opens/closes with keyboard while restoring focus", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { CustomMenu } = await loadDropdowns();
  const openCallbacks = [];
  const closeCallbacks = [];
  const childRef = React.createRef();
  const trigger = React.createElement("button", { ref: childRef, "aria-label": "More actions" }, "Actions");
  await environment.render(
    React.createElement(
      CustomMenu,
      {
        customButton: trigger,
        customButtonAsChild: true,
        ariaLabel: "More actions",
        menuButtonOnClick: () => openCallbacks.push("opened"),
        onMenuClose: () => closeCallbacks.push("closed"),
      },
      React.createElement(CustomMenu.MenuItem, null, "Open item")
    )
  );

  const triggerElement = environment.container.querySelector("button");
  assert.equal(environment.container.querySelectorAll("button").length, 1);
  assert.equal(childRef.current, triggerElement);
  assert.equal(triggerElement.getAttribute("aria-label"), "More actions");

  await React.act(() => keyDown(triggerElement, "Enter"));
  assert.deepEqual(openCallbacks, ["opened"]);
  const menuItem = [...document.querySelectorAll("button")].find((button) => button.textContent.includes("Open item"));
  assert.ok(menuItem, "Enter opens a visible menu item");

  await React.act(() => menuItem.focus());
  await React.act(() => keyDown(menuItem, "Escape"));
  assert.deepEqual(closeCallbacks, ["closed"]);
  assert.equal(document.querySelectorAll("button").length, 1);

  await React.act(() => keyDown(triggerElement, " "));
  assert.equal(openCallbacks.length, 2);
  assert.ok([...document.querySelectorAll("button")].some((button) => button.textContent.includes("Open item")));
  const spaceMenuItem = [...document.querySelectorAll("button")].find((button) =>
    button.textContent.includes("Open item")
  );
  await React.act(() => spaceMenuItem.focus());
  await React.act(() => keyDown(spaceMenuItem, "Escape"));

  await React.act(() => keyDown(triggerElement, "ArrowDown"));
  assert.ok([...document.querySelectorAll("button")].some((button) => button.textContent.includes("Open item")));
  const arrowDownMenuItem = [...document.querySelectorAll("button")].find((button) =>
    button.textContent.includes("Open item")
  );
  await React.act(() => arrowDownMenuItem.focus());
  await React.act(() => keyDown(arrowDownMenuItem, "Escape"));
  assert.equal(openCallbacks.length, 3);
  assert.equal(closeCallbacks.length, 3);
});

test("CustomMenu keeps a disabled child disabled and retains the text-content compatibility trigger", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { CustomMenu } = await loadDropdowns();
  await environment.render(
    React.createElement(
      React.Fragment,
      null,
      React.createElement(
        CustomMenu,
        {
          customButton: React.createElement("button", { disabled: true }, "Child disabled"),
          customButtonAsChild: true,
        },
        React.createElement(CustomMenu.MenuItem, null, "Unavailable")
      ),
      React.createElement(CustomMenu, {
        disabled: true,
        customButton: React.createElement("button", null, "Parent disabled"),
        customButtonAsChild: true,
      }),
      React.createElement(
        CustomMenu,
        { customButton: "Text trigger" },
        React.createElement(CustomMenu.MenuItem, null, "Compatibility item")
      )
    )
  );

  const buttons = [...environment.container.querySelectorAll("button")];
  assert.equal(buttons.length, 3);
  assert.equal(buttons[0].disabled, true);
  assert.equal(buttons[0].textContent, "Child disabled");
  assert.equal(buttons[1].disabled, true);
  assert.equal(buttons[1].textContent, "Parent disabled");
  assert.equal(buttons[2].textContent, "Text trigger");

  await React.act(async () => buttons[0].click());
  assert.equal(document.querySelectorAll("button").length, 3);
  await React.act(async () => buttons[2].click());
  assert.ok(
    [...document.querySelectorAll("button")].some((button) => button.textContent.includes("Compatibility item"))
  );
});

test("CustomSelect and CustomSearchSelect keep one trigger and call onOpen after opening", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { CustomSelect, CustomSearchSelect } = await loadDropdowns();
  const opened = [];
  const openedAfterRerender = [];
  const closed = [];
  const selectChanges = [];
  const searchChanges = [];
  const selectTrigger = React.createElement("button", null, "Select trigger");
  const searchTrigger = React.createElement("button", null, "Search trigger");
  const renderDropdowns = (onOpen) =>
    React.createElement(
      React.Fragment,
      null,
      React.createElement(
        CustomSelect,
        {
          customButton: selectTrigger,
          customButtonAsChild: true,
          value: null,
          onChange: (value) => selectChanges.push(value),
        },
        React.createElement(CustomSelect.Option, { value: "one" }, "Select option")
      ),
      React.createElement(CustomSearchSelect, {
        customButton: searchTrigger,
        customButtonAsChild: true,
        value: null,
        options: [{ value: "two", content: "Search option", query: "search option" }],
        onChange: (value) => searchChanges.push(value),
        onOpen,
        onClose: () => closed.push("closed"),
      })
    );
  await environment.render(renderDropdowns(() => opened.push("open")));

  const buttons = [...environment.container.querySelectorAll("button")];
  assert.equal(buttons.length, 2);
  assert.equal(buttons[0].textContent, "Select trigger");
  assert.equal(buttons[1].textContent, "Search trigger");
  await React.act(async () => buttons[0].click());
  assert.ok(document.body.textContent.includes("Select option"));
  const selectOption = [...document.querySelectorAll("[role=option]")].find((option) =>
    option.textContent.includes("Select option")
  );
  await React.act(async () => {
    selectOption.click();
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  assert.deepEqual(selectChanges, ["one"]);
  assert.equal(document.body.textContent.includes("Select option"), false);
  const search = environment.container.querySelectorAll("button")[1];
  await React.act(async () => search.click());
  assert.ok(document.body.textContent.includes("Search option"));
  assert.deepEqual(opened, ["open"]);
  await environment.render(renderDropdowns(() => openedAfterRerender.push("open")));
  assert.deepEqual(opened, ["open"]);
  assert.deepEqual(openedAfterRerender, []);

  const searchOption = [...document.querySelectorAll("[role=option]")].find((option) =>
    option.textContent.includes("Search option")
  );
  await React.act(async () => keyDown(searchOption, "Escape"));
  assert.deepEqual(opened, ["open"]);
  assert.deepEqual(closed, ["closed"]);

  await React.act(async () => environment.container.querySelectorAll("button")[1].click());
  const reopenedSearchOption = [...document.querySelectorAll("[role=option]")].find((option) =>
    option.textContent.includes("Search option")
  );
  await React.act(async () => reopenedSearchOption.click());
  assert.deepEqual(searchChanges, ["two"]);
  assert.deepEqual(closed, ["closed", "closed"]);
});

test("CustomSelect and CustomSearchSelect honor disabled state from either the wrapper or child trigger", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { CustomSelect, CustomSearchSelect } = await loadDropdowns();
  await environment.render(
    React.createElement(
      React.Fragment,
      null,
      React.createElement(
        CustomSelect,
        {
          customButton: React.createElement("button", null, "Select wrapper disabled"),
          customButtonAsChild: true,
          disabled: true,
          value: null,
          onChange() {},
        },
        React.createElement(CustomSelect.Option, { value: "one" }, "Select option")
      ),
      React.createElement(
        CustomSelect,
        {
          customButton: React.createElement("button", { disabled: true }, "Select child disabled"),
          customButtonAsChild: true,
          value: null,
          onChange() {},
        },
        React.createElement(CustomSelect.Option, { value: "two" }, "Another select option")
      ),
      React.createElement(CustomSearchSelect, {
        customButton: React.createElement("button", null, "Search wrapper disabled"),
        customButtonAsChild: true,
        disabled: true,
        value: null,
        options: [{ value: "three", content: "Search option", query: "search" }],
        onChange() {},
      }),
      React.createElement(CustomSearchSelect, {
        customButton: React.createElement("button", { disabled: true }, "Search child disabled"),
        customButtonAsChild: true,
        value: null,
        options: [{ value: "four", content: "Another search option", query: "search" }],
        onChange() {},
      })
    )
  );

  const triggers = [...environment.container.querySelectorAll("button")];
  assert.deepEqual(
    triggers.map((trigger) => [trigger.textContent, trigger.disabled]),
    [
      ["Select wrapper disabled", true],
      ["Select child disabled", true],
      ["Search wrapper disabled", true],
      ["Search child disabled", true],
    ]
  );
  await React.act(() => triggers[0].click());
  await React.act(() => triggers[1].click());
  await React.act(() => triggers[2].click());
  await React.act(() => triggers[3].click());
  assert.equal(document.querySelector("[role=option]"), null);
});
