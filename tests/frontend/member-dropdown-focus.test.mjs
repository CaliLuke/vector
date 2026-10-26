import { createRouterFixture } from "./helpers/router.mjs";
import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { fileURLToPath, pathToFileURL } from "node:url";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { clickWithPointer } from "./helpers/pointer.mjs";

globalThis.memberDropdownTestReact = React;
const repositoryRoot = fileURLToPath(new URL("../..", import.meta.url));
const comboBoxUrl = pathToFileURL(`${repositoryRoot}/packages/ui/src/dropdowns/combo-box.tsx`).href;
const modalCoreUrl = pathToFileURL(`${repositoryRoot}/packages/ui/src/modals/modal-core.tsx`).href;
registerWebModuleHooks();

const packageStubs = {
  "@plane/i18n": "export const useTranslation = () => ({ t: (key) => key });",
  "@plane/propel/icons": `export const CheckIcon = () => null;
    export const SearchIcon = () => null;
    export const SuspendedUserIcon = () => null;
    export const ChevronDownIcon = () => null;`,
  "@plane/propel/pill": `export const EPillSize = { XS: "xs" };
    export const EPillVariant = { DEFAULT: "default" };
    export const Pill = () => null;`,
  "@plane/propel/button": `export const getButtonStyling = () => "";
    export const Button = (props) => globalThis.memberDropdownTestReact.createElement("button", props, props.children);`,
  "@plane/propel/tooltip": "export const Tooltip = ({ children }) => children;",
  "@plane/ui": `export { ComboDropDown, ComboSearchInput } from ${JSON.stringify(comboBoxUrl)};
    export const Avatar = () => null;
    export const AvatarGroup = ({ children }) => children;`,
  "@plane/utils/common":
    'export const cn = (...values) => values.flatMap((value) =>\n      typeof value === "string" ? [value] : value && typeof value === "object" ? Object.keys(value).filter((key) => value[key]) : []\n    ).join(" ");',
  "@plane/utils/file": "export const getFileURL = (value) => value;",
  "@plane/utils/array": "export const sortByCurrentUserThenSelected = (options) => options;",
  "react-popper": "export const usePopper = () => ({ styles: { popper: {} }, attributes: { popper: {} } });",
  "@/hooks/use-current-user": `export const useCurrentUser = () => ({ data: { id: "current-user" } });`,
  "@/hooks/use-platform-os": `export const usePlatformOS = () => ({ isMobile: false });`,
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (Object.hasOwn(packageStubs, specifier)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(packageStubs[specifier])}` };
    }
    if (specifier === "../buttons" && context.parentURL?.endsWith("/dropdowns/member/base.tsx")) {
      const source = `export const DropdownButton = ({ children }) => children;`;
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(source)}` };
    }
    if (specifier === "./avatar" && context.parentURL?.endsWith("/dropdowns/member/base.tsx")) {
      const source = `export const ButtonAvatars = () => null;`;
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(source)}` };
    }
    if (specifier === "../utils" && context.parentURL?.endsWith("/modals/modal-core.tsx")) {
      const source = `export const cn = (...values) => values.filter(Boolean).join(" ");`;
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(source)}` };
    }
    if (specifier === "../form-fields/root" && context.parentURL?.endsWith("/modals/modal-core.tsx")) {
      const source = `export const Form = ({ children }) => children;`;
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(source)}` };
    }
    return nextResolve(specifier, context);
  },
});

let MemberDropdownBase;
async function loadMemberDropdown() {
  if (!MemberDropdownBase) {
    ({ MemberDropdownBase } = await import("../../apps/web/core/components/dropdowns/member/base.tsx"));
  }
  return MemberDropdownBase;
}

function memberDropdownProps(value = [], onChange = () => {}) {
  return {
    getUserDetails: (id) => ({ id, display_name: "You", first_name: "You", last_name: "" }),
    getIsUserSuspended: () => false,
    isRosterLoading: false,
    isRosterError: false,
    onRosterRetry: () => {},
    memberIds: ["current-user"],
    multiple: true,
    value,
    onChange,
    placeholder: "assignees",
    buttonVariant: "border-with-text",
    workspaceSlug: "workspace",
  };
}

async function renderMemberDropdown(environment) {
  const routerFixture = createRouterFixture("/workspace/");
  const Component = await loadMemberDropdown();
  function Harness() {
    const [value, setValue] = React.useState([]);
    return React.createElement(Component, memberDropdownProps(value, setValue));
  }
  await routerFixture.render(environment, React.createElement(Harness));
  return environment.container.querySelector("button");
}

async function openMemberDropdown(environment, trigger) {
  await clickWithPointer(trigger);
  const nativeOpen = trigger.getAttribute("aria-expanded");
  const input = document.querySelector('input[placeholder="search"]');
  assert.ok(input, "opening the member dropdown should mount its search input");
  assert.equal(input.getAttribute("role"), "combobox");
  assert.equal(nativeOpen, "true", "Headless UI should open on the native pointer sequence");
  assert.equal(document.activeElement, input, "opening the member dropdown should focus search");
  return input;
}

function pressKey(element, key) {
  element.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true }));
}

function setInputValue(input, value) {
  Reflect.apply(Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set, input, [value]);
  input.dispatchEvent(new Event("input", { bubbles: true }));
}

await test("member picker Escape after selecting You closes and restores trigger focus", async () => {
  const environment = await createReactDomTestEnvironment();
  const trigger = await renderMemberDropdown(environment);
  assert.ok(trigger.closest('[role="group"]'), "member picker preserves its group semantics");
  const input = await openMemberDropdown(environment, trigger);
  const you = [...document.querySelectorAll('[role="option"]')].find((option) => option.textContent.includes("you"));
  assert.ok(you, "current user should appear in the member options");
  await React.act(async () => you.click());
  assert.ok(document.querySelector('[role="listbox"]'), "multi-select should remain open after selection");
  assert.equal(document.activeElement, input);

  await React.act(() => pressKey(input, "Escape"));
  const closed = !document.querySelector('[role="listbox"]');
  const focusRestored = document.activeElement === trigger;
  await environment.unmount();
  assert.ok(closed, "Escape should close the picker");
  assert.ok(focusRestored, "Escape should restore focus to the picker trigger");
});

await test("member picker Escape does not close its containing modal", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(async () => {
    await environment.unmount();
  });
  const routerFixture = createRouterFixture("/workspace/");
  const Component = await loadMemberDropdown();
  const { ModalCore } = await import(modalCoreUrl);

  function Harness() {
    const [isOpen, setIsOpen] = React.useState(true);
    const [value, setValue] = React.useState([]);
    return React.createElement(
      React.Fragment,
      null,
      React.createElement("output", { "data-testid": "modal-open" }, String(isOpen)),
      React.createElement(
        "form",
        {
          onKeyDown: (event) => {
            if (event.key !== "Escape" || event.defaultPrevented) return;
            event.preventDefault();
            setIsOpen(false);
          },
        },
        React.createElement(
          ModalCore,
          { isOpen, handleClose: () => setIsOpen(false) },
          React.createElement(Component, memberDropdownProps(value, setValue))
        )
      )
    );
  }

  await routerFixture.render(environment, React.createElement(Harness));
  const trigger = [...document.querySelectorAll("button")].find((button) => button.textContent.includes("assignees"));
  assert.ok(trigger, "the actual dialog should mount the member picker trigger");
  await openMemberDropdown(environment, trigger);
  const you = [...document.querySelectorAll('[role="option"]')].find((option) => option.textContent.includes("you"));
  assert.ok(you, "current user should appear in the member options");
  await React.act(async () => you.click());

  const input = document.querySelector('input[placeholder="search"]');
  await React.act(() => pressKey(input, "Escape"));

  assert.equal(
    environment.container.querySelector('[data-testid="modal-open"]')?.textContent,
    "true",
    "Escape should be consumed by the picker before the modal"
  );
  await environment.unmount();
});

await test("member picker clears a nonempty query before Escape closes it", async () => {
  const environment = await createReactDomTestEnvironment();
  const trigger = await renderMemberDropdown(environment);
  const input = await openMemberDropdown(environment, trigger);
  await React.act(() => setInputValue(input, "You"));
  assert.equal(input.value, "You");

  await React.act(() => pressKey(input, "Escape"));
  const clearedFirst =
    input.value === "" && !!document.querySelector('[role="listbox"]') && document.activeElement === input;

  await React.act(() => pressKey(input, "Escape"));
  const closed = !document.querySelector('[role="listbox"]');
  const focusRestored = document.activeElement === trigger;
  await environment.unmount();
  assert.ok(clearedFirst, "first Escape should clear search and keep focus/menu open");
  assert.ok(closed, "second Escape should close the picker");
  assert.ok(focusRestored, "second Escape should restore focus to the picker trigger");
});

await test("member picker outside click closes without stealing outside focus", async () => {
  const environment = await createReactDomTestEnvironment();
  const trigger = await renderMemberDropdown(environment);
  await openMemberDropdown(environment, trigger);
  const outside = document.createElement("button");
  outside.textContent = "Outside";
  environment.container.append(outside);
  await React.act(() => outside.focus());

  await React.act(() => outside.dispatchEvent(new MouseEvent("mousedown", { bubbles: true })));

  const closed = !document.querySelector('[role="listbox"]');
  const focusStayedOutside = document.activeElement === outside;
  await environment.unmount();
  assert.ok(closed);
  assert.ok(focusStayedOutside, "outside click should not move focus back to the dropdown trigger");
});

await test("member picker error offers a retry action", async () => {
  const environment = await createReactDomTestEnvironment();
  let retryCount = 0;
  const routerFixture = createRouterFixture("/workspace/");
  const Component = await loadMemberDropdown();
  await routerFixture.render(
    environment,
    React.createElement(Component, {
      getUserDetails: () => undefined,
      getIsUserSuspended: () => false,
      isRosterLoading: false,
      isRosterError: true,
      onRosterRetry: () => {
        retryCount += 1;
      },
      memberIds: [],
      multiple: true,
      value: [],
      onChange: () => {},
      placeholder: "assignees",
      buttonVariant: "border-with-text",
    })
  );

  const trigger = environment.container.querySelector("button");
  await React.act(async () => trigger.click());
  const retry = [...document.querySelectorAll("button")].find((button) => button.textContent === "retry");
  assert.ok(retry, "a failed roster read should expose a retry button");
  await React.act(async () => retry.click());
  assert.equal(retryCount, 1);
  await environment.unmount();
});
