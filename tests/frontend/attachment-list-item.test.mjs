import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const customMenuUrl = new URL("../../packages/ui/src/dropdowns/custom-menu.tsx", import.meta.url).href;
const packageStubs = {
  "mobx-react": "export const observer = component => component;",
  "@plane/i18n": "export const useTranslation = () => ({ t: key => key });",
  "@plane/propel/icons":
    "export const TrashIcon = () => null; export const ChevronDownIcon = () => null; export const ChevronRightIcon = () => null;",
  "@plane/propel/tooltip": `export { Tooltip } from ${JSON.stringify(new URL("../../packages/propel/src/tooltip/root.tsx", import.meta.url).href)};`,
  "@plane/types": "export const EIssueServiceType = { ISSUES: 'issues' };",
  "@plane/ui": `export { CustomMenu } from ${JSON.stringify(customMenuUrl)};`,
  "@plane/utils":
    "export const convertBytesToSize = value => `${value} bytes`; export const getFileExtension = value => value.split('.').pop(); export const getFileName = value => value.split('.').slice(0, -1).join('.'); export const getFileURL = value => value; export const renderFormattedDate = () => 'today';",
  "@/components/dropdowns/member/avatar": "export const ButtonAvatars = () => null;",
  "@/components/icons": "export const getFileIcon = () => null;",
  "@/hooks/store/use-issue-detail": `export const useIssueDetail = () => ({ attachment: { getAttachmentById: () => ({ attributes: { name: 'report.pdf', size: 68 }, asset_url: 'https://files.test/report.pdf', created_by: null, updated_at: '2026-10-19T00:00:00Z' }) }, toggleDeleteAttachmentModal: value => globalThis.__attachmentCalls.push(['delete', value]) });`,
  "@/hooks/store/use-member": "export const useMember = () => ({ getUserDetails: () => undefined });",
  "@/hooks/use-platform-os": "export const usePlatformOS = () => ({ isMobile: false });",
  "@plane/hooks": "export const useOutsideClickDetector = () => {};",
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

globalThis.__attachmentCalls = [];
async function renderAttachment(environment, props) {
  const { IssueAttachmentsListItem } =
    await import("../../apps/web/core/components/issues/attachment/attachment-list-item.tsx");
  await environment.render(React.createElement(IssueAttachmentsListItem, props));
}

async function waitFor(condition, message) {
  const deadline = Date.now() + 300;
  await new Promise((resolve) => {
    const poll = () => {
      if (condition() || Date.now() >= deadline) resolve();
      else setTimeout(poll, 5);
    };
    poll();
  });
  assert.ok(condition(), message);
}

test("attachment row keeps file opening and its real menu in separate buttons", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const errors = [];
  const originalConsoleError = console.error;
  console.error = (...args) => errors.push(args.join(" "));
  t.after(() => {
    console.error = originalConsoleError;
  });

  const opened = [];
  const testWindow = window;
  const originalOpen = testWindow.open;
  testWindow.open = (...args) => opened.push(args);
  t.after(() => {
    testWindow.open = originalOpen;
  });

  await renderAttachment(environment, { attachmentId: "attachment-1" });

  const fileButton = [...environment.container.querySelectorAll("button")].find((button) =>
    button.textContent.includes("report.pdf")
  );
  const menuButton = environment.container.querySelector("button[aria-haspopup='menu']");
  assert.ok(fileButton);
  assert.ok(menuButton);
  assert.equal(fileButton.contains(menuButton), false);
  assert.equal(fileButton.querySelector("button"), null);
  assert.equal(
    errors.some((message) => message.includes("validateDOMNesting")),
    false,
    "the row should not render nested buttons"
  );

  await React.act(async () => fileButton.click());
  assert.deepEqual(opened, [["https://files.test/report.pdf", "_blank"]]);

  menuButton.focus();
  await React.act(async () => {
    menuButton.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true }));
  });
  assert.equal(menuButton.getAttribute("aria-expanded"), "true", "Enter should open the menu");
  assert.ok(document.body.textContent.includes("common.actions.delete"));
  const deleteItem = [...document.querySelectorAll("[role='menuitem']")].find((item) =>
    item.textContent.includes("common.actions.delete")
  );
  assert.ok(deleteItem);
  const menu = document.querySelector("[role='menu']");
  const escapeTarget = document.activeElement ?? menu;
  await React.act(async () => {
    escapeTarget.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
  });
  await waitFor(() => menuButton.getAttribute("aria-expanded") === "false", "Escape should close the menu");
  assert.equal(document.querySelector("[role='menuitem']"), null);
  await waitFor(() => document.activeElement === menuButton, "Escape should return focus to the menu trigger");
});

test("attachment menu keeps its delete action when opened by pointer", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());

  await renderAttachment(environment, { attachmentId: "attachment-1" });
  const menuButton = environment.container.querySelector("button[aria-haspopup='menu']");
  await React.act(async () =>
    menuButton.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, detail: 1 }))
  );
  const deleteItem = [...document.querySelectorAll("[role='menuitem']")].find((item) =>
    item.textContent.includes("common.actions.delete")
  );
  assert.ok(deleteItem);
  await React.act(async () => deleteItem.querySelector("button").click());
  assert.deepEqual(globalThis.__attachmentCalls, [["delete", "attachment-1"]]);
});

test("disabled attachment menu does not open", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());

  await renderAttachment(environment, { attachmentId: "attachment-1", disabled: true });

  const menuButton = environment.container.querySelector("button[aria-haspopup='menu']");
  assert.ok(menuButton);
  assert.equal(menuButton.disabled, true);
  await React.act(async () => menuButton.click());
  assert.equal(document.querySelector("[role='menuitem']"), null);
});
