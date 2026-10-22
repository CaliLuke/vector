import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { fileURLToPath, pathToFileURL } from "node:url";
import { React, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

const repositoryRoot = fileURLToPath(new URL("../..", import.meta.url));
const comboBoxUrl = pathToFileURL(`${repositoryRoot}/packages/ui/src/dropdowns/combo-box.tsx`).href;
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
  "@plane/propel/button": 'export const getButtonStyling = () => "";',
  "@plane/propel/tooltip": "export const Tooltip = ({ children }) => children;",
  "@plane/ui": `export { ComboDropDown } from ${JSON.stringify(comboBoxUrl)};
    export const Avatar = () => null;
    export const AvatarGroup = ({ children }) => children;`,
  "@plane/utils": `export const cn = (...values) => values.flatMap((value) =>
      typeof value === "string" ? [value] : value && typeof value === "object" ? Object.keys(value).filter((key) => value[key]) : []
    ).join(" ");
    export const getFileURL = (value) => value;
    export const sortByCurrentUserThenSelected = (options) => options;`,
  "mobx-react": "export const observer = (component) => component;",
  "next/navigation": 'export const useParams = () => ({ workspaceSlug: "workspace" });',
  "react-popper": "export const usePopper = () => ({ styles: { popper: {} }, attributes: { popper: {} } });",
  "@/hooks/store/use-member": `export const useMember = () => ({ workspace: { isUserSuspended: () => false } });`,
  "@/hooks/store/user": `export const useUser = () => ({ data: { id: "current-user" } });`,
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

async function renderMemberDropdown(environment) {
  const Component = await loadMemberDropdown();
  function Harness() {
    const [value, setValue] = React.useState([]);
    return React.createElement(Component, {
      getUserDetails: (id) => ({ id, display_name: "You", first_name: "You", last_name: "" }),
      memberIds: ["current-user"],
      multiple: true,
      value,
      onChange: setValue,
      placeholder: "assignees",
      buttonVariant: "border-with-text",
    });
  }
  await environment.render(React.createElement(Harness));
  return environment.container.querySelector("button");
}

async function openMemberDropdown(environment, trigger) {
  await React.act(async () => trigger.click());
  const input = document.querySelector('input[placeholder="search"]');
  assert.ok(input, "opening the member dropdown should mount its search input");
  assert.equal(document.activeElement, input, "opening the member dropdown should focus search");
  return input;
}

function pressKey(element, key) {
  element.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true }));
}

function setInputValue(input, value) {
  const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set;
  setter.call(input, value);
  input.dispatchEvent(new Event("input", { bubbles: true }));
}

test("member picker Escape after selecting You closes and restores trigger focus", async () => {
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

test("member picker clears a nonempty query before Escape closes it", async () => {
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

test("member picker outside click closes without stealing outside focus", async () => {
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
