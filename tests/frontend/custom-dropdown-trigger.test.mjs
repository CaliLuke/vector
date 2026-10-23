import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import { after, afterEach, test } from "node:test";
import { React, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { createModuleStub, resolvePackageStub } from "./helpers/module-stubs.mjs";
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
      return createModuleStub(
        "export const cn = (...values) => values.filter(value => typeof value === 'string').join(' ');"
      );
    }
    return resolvePackageStub(specifier, packageStubs, context, nextResolve);
  },
});

const environment = await createReactDomTestEnvironment();

afterEach(async () => {
  await environment.render(null);
});

after(async () => {
  await environment.unmount();
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
  await React.act(async () => {
    const useNativeClick = keyDown(element, key);
    // jsdom does not produce the native click for an uncancelled Enter/Space key.
    if (useNativeClick) element.click();
    await waitForAnimationFramesInCurrentAct();
  });
}

async function activateOption(option) {
  await React.act(async () => {
    option.dispatchEvent(new MouseEvent("mousedown", { button: 0, bubbles: true, cancelable: true }));
    option.click();
  });
}

async function waitForCondition(condition, message, timeoutMs = 2_000) {
  const deadline = performance.now() + timeoutMs;
  await React.act(async () => {
    await new Promise((resolve) => {
      const check = () => (condition() || performance.now() >= deadline ? resolve() : setTimeout(check, 10));
      check();
    });
  });
  assert.ok(condition(), message);
}

function waitForAnimationFramesInCurrentAct(frameCount = 2) {
  return new Promise((resolve) => {
    let frames = 0;
    const flushNextFrame = () => {
      frames += 1;
      if (frames === frameCount) resolve();
      else requestAnimationFrame(flushNextFrame);
    };
    requestAnimationFrame(flushNextFrame);
  });
}

async function actWithAnimationFrames(action) {
  await React.act(async () => {
    action();
    await waitForAnimationFramesInCurrentAct();
  });
}

function assertNoConsoleErrors(t) {
  const errors = [];
  const originalConsoleError = console.error;
  const originalConsoleWarn = console.warn;
  console.error = (...args) => errors.push(args);
  console.warn = (...args) => errors.push(args);
  t.after(() => {
    console.error = originalConsoleError;
    console.warn = originalConsoleWarn;
    assert.deepEqual(errors, [], "the interaction should not log React or browser errors");
  });
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

async function renderDropdown(kind, callbacks = {}) {
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
    assertNoConsoleErrors(t);
    const trigger = await renderDropdown(kind);

    if (kind === "menu") await actWithAnimationFrames(() => trigger.click());
    else await React.act(() => trigger.click());
    assert.equal(trigger.getAttribute("aria-expanded"), "true");
    assert.ok(document.querySelector("[role=menu], [role=listbox]"));

    if (kind === "menu") await actWithAnimationFrames(() => trigger.click());
    else await React.act(() => trigger.click());
    assert.equal(trigger.getAttribute("aria-expanded"), "false");
    assert.ok(!document.querySelector("[role=menu], [role=listbox]"), "a second activation should close the popup");

    if (kind === "menu") await actWithAnimationFrames(() => trigger.click());
    else await React.act(() => trigger.click());
    assert.equal(trigger.getAttribute("aria-expanded"), "true");
    assert.ok(document.querySelector("[role=menu], [role=listbox]"));
  });

  for (const key of ["Enter", " ", "ArrowDown", "ArrowUp"]) {
    test(`${kind} opens once with ${JSON.stringify(key)} and closes with Escape`, async (t) => {
      assertNoConsoleErrors(t);
      const trigger = await renderDropdown(kind);

      if (kind === "menu") await actWithAnimationFrames(() => trigger.focus());
      else await React.act(() => trigger.focus());
      await activateWithKeyboard(trigger, key);
      assert.equal(trigger.getAttribute("aria-expanded"), "true");
      await waitForCondition(
        () => document.querySelector("[role=menu], [role=listbox]"),
        "keyboard activation should render the popup"
      );
      const options = document.querySelector("[role=menu], [role=listbox]");
      assert.ok(options);

      const escapeTarget = kind === "menu" ? options : kind === "search" ? options.querySelector("input") : trigger;
      await waitForFocus(escapeTarget, "keyboard opening should focus the active menu or search control");
      if (kind === "menu") await actWithAnimationFrames(() => keyDown(escapeTarget, "Escape"));
      else await React.act(() => keyDown(escapeTarget, "Escape"));
      assert.equal(trigger.getAttribute("aria-expanded"), "false");
      await waitForCondition(
        () => !document.querySelector("[role=menu], [role=listbox]"),
        "Escape should close the popup"
      );
      await waitForFocus(trigger);
    });
  }
}

test("CustomMenu composes a button trigger and opens/closes with keyboard while restoring focus", async (t) => {
  assertNoConsoleErrors(t);
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

  await activateWithKeyboard(triggerElement, "Enter");
  assert.deepEqual(openCallbacks, ["opened"]);
  const menuItem = [...document.querySelectorAll("button")].find((button) => button.textContent.includes("Open item"));
  assert.ok(menuItem, "Enter opens a visible menu item");

  await actWithAnimationFrames(() => menuItem.focus());
  await actWithAnimationFrames(() => keyDown(menuItem, "Escape"));
  assert.deepEqual(closeCallbacks, ["closed"]);
  assert.equal(document.querySelectorAll("button").length, 1);

  await activateWithKeyboard(triggerElement, " ");
  assert.equal(openCallbacks.length, 2);
  assert.ok([...document.querySelectorAll("button")].some((button) => button.textContent.includes("Open item")));
  const spaceMenuItem = [...document.querySelectorAll("button")].find((button) =>
    button.textContent.includes("Open item")
  );
  await actWithAnimationFrames(() => spaceMenuItem.focus());
  await actWithAnimationFrames(() => keyDown(spaceMenuItem, "Escape"));

  await activateWithKeyboard(triggerElement, "ArrowDown");
  assert.ok([...document.querySelectorAll("button")].some((button) => button.textContent.includes("Open item")));
  const arrowDownMenuItem = [...document.querySelectorAll("button")].find((button) =>
    button.textContent.includes("Open item")
  );
  await actWithAnimationFrames(() => arrowDownMenuItem.focus());
  await actWithAnimationFrames(() => keyDown(arrowDownMenuItem, "Escape"));
  assert.equal(openCallbacks.length, 3);
  assert.equal(closeCallbacks.length, 3);
});

test("CustomMenu keeps a disabled child disabled and retains the text-content compatibility trigger", async (t) => {
  assertNoConsoleErrors(t);
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
  assertNoConsoleErrors(t);
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
  await activateOption(selectOption);
  await waitForCondition(() => selectChanges.length > 0, "selecting an option should notify onChange");
  assert.deepEqual(selectChanges, ["one"]);
  assert.equal(document.body.textContent.includes("Select option"), false);
  const search = environment.container.querySelectorAll("button")[1];
  await React.act(async () => search.click());
  assert.ok(document.body.textContent.includes("Search option"));
  await waitForCondition(() => opened.length > 0, "opening the search dropdown should call onOpen");
  assert.deepEqual(opened, ["open"]);
  await environment.render(renderDropdowns(() => openedAfterRerender.push("open")));
  assert.deepEqual(opened, ["open"]);
  assert.deepEqual(openedAfterRerender, []);

  const searchInput = document.querySelector("input[placeholder='Search']");
  assert.ok(searchInput, "the searchable dropdown should focus its input");
  await React.act(async () => {
    searchInput.focus();
    keyDown(searchInput, "Escape");
  });
  await waitForCondition(() => closed.length > 0, "closing the search dropdown should call onClose");
  assert.deepEqual(opened, ["open"]);
  assert.deepEqual(closed, ["closed"]);

  const searchTriggerAfterEscape = environment.container.querySelectorAll("button")[1];
  await React.act(async () => searchTriggerAfterEscape.click());
  await waitForCondition(
    () => searchTriggerAfterEscape.getAttribute("aria-expanded") === "true",
    "the search trigger should reopen the Headless UI Combobox"
  );
  await waitForCondition(
    () => document.querySelectorAll("[role=option]").length > 0,
    "reopening the search dropdown should render its option"
  );
  const reopenedSearchOption = [...document.querySelectorAll("[role=option]")].find((option) =>
    option.textContent.includes("Search option")
  );
  await activateOption(reopenedSearchOption);
  await waitForCondition(() => searchChanges.length > 0, "selecting a search option should notify onChange");
  assert.deepEqual(searchChanges, ["two"]);
  assert.deepEqual(closed, ["closed", "closed"]);
});

void test("CustomSearchSelect defaultOpen synchronizes Headless UI state and closes with callbacks", async (t) => {
  assertNoConsoleErrors(t);
  const { CustomSearchSelect } = await loadDropdowns();
  const defaultOpenEvents = [];
  await environment.render(
    React.createElement(CustomSearchSelect, {
      customButton: React.createElement("button", null, "Search trigger"),
      customButtonAsChild: true,
      defaultOpen: true,
      value: null,
      options: [{ value: "one", content: "Option one", query: "option one" }],
      onChange() {},
      onOpen: () => defaultOpenEvents.push("open"),
      onClose: () => defaultOpenEvents.push("close"),
    })
  );

  const trigger = environment.container.querySelector("button");
  await waitForCondition(
    () => trigger.getAttribute("aria-expanded") === "true",
    "defaultOpen should synchronize Headless UI's expanded state"
  );
  assert.deepEqual(defaultOpenEvents, ["open"]);

  await React.act(async () => trigger.click());
  await waitForCondition(
    () => trigger.getAttribute("aria-expanded") === "false",
    "the first trigger click should close the default-open dropdown"
  );
  assert.deepEqual(defaultOpenEvents, ["open", "close"]);

  await React.act(async () => trigger.click());
  await waitForCondition(
    () => trigger.getAttribute("aria-expanded") === "true",
    "the trigger should reopen the dropdown"
  );
  const defaultOpenSearchInput = document.querySelector("input[placeholder='Search']");
  assert.ok(defaultOpenSearchInput, "the reopened dropdown should expose its search input");
  await React.act(async () => {
    defaultOpenSearchInput.focus();
    keyDown(defaultOpenSearchInput, "Escape");
  });
  await waitForCondition(
    () => trigger.getAttribute("aria-expanded") === "false",
    "Escape should close the dropdown's Headless UI state"
  );
  assert.deepEqual(defaultOpenEvents, ["open", "close", "open", "close"]);
});

test("CustomSelect and CustomSearchSelect honor disabled state from either the wrapper or child trigger", async (t) => {
  assertNoConsoleErrors(t);
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

void test("CustomSelect preserves an explicit null-valued option", async (t) => {
  assertNoConsoleErrors(t);
  const { CustomSelect } = await loadDropdowns();
  const changes = [];
  await environment.render(
    React.createElement(
      CustomSelect,
      {
        customButton: React.createElement("button", null, "Select trigger"),
        customButtonAsChild: true,
        value: "selected",
        onChange: (value) => changes.push(value),
      },
      React.createElement(CustomSelect.Option, { value: null }, "No value")
    )
  );

  await React.act(async () => environment.container.querySelector("button").click());
  await waitForCondition(() => document.querySelector("[role=option]"), "the null-valued option should render");
  const option = document.querySelector("[role=option]");
  await activateOption(option);

  assert.deepEqual(changes, [null]);
  await waitForFocus(environment.container.querySelector("button"));
  await React.act(waitForAnimationFramesInCurrentAct);
});
