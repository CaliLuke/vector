import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createReactDomTestEnvironment, React } from "./helpers/react-dom.mjs";

registerWebModuleHooks();
const emojiPickerSource = new URL("../../packages/propel/src/emoji-icon-picker/index.ts", import.meta.url).href;
const pageMenuStubs = {
  "@/hooks/use-app-router": `export const useAppRouter = () => ({ push() {} });`,
  "@/hooks/use-page-filters": `export const usePageFilters = () => ({
    isFullWidth: globalThis.__pageActionsTest.fullWidth,
    handleFullWidth: value => globalThis.__pageActionsTest.fullWidthChanges.push(value),
    isStickyToolbarEnabled: globalThis.__pageActionsTest.stickyToolbar,
    handleStickyToolbar: value => globalThis.__pageActionsTest.stickyChanges.push(value),
  });`,
  "@/hooks/use-query-params": `export const useQueryParams = () => ({ updateQueryParams: () => "" });`,
  "@/hooks/use-page-operations": `export const usePageOperations = () => ({ pageOperations: {
    toggleLock() {}, toggleAccess() {}, openInNewTab() {}, copyLink() {}, duplicate() {},
    toggleArchive() {}, move() {}, copyMarkdownToClipboard() {},
  } });`,
  "@/hooks/use-page-flag": `export const usePageFlag = () => ({ isMovePageEnabled: false });`,
  "@/components/pages/modals/delete-page-modal": `export function DeletePageModal() { return null; }`,
  "@/components/pages/modals/export-page-modal": `export function ExportPageModal() { return null; }`,
  "next/navigation": `export const useParams = () => ({ workspaceSlug: "workspace" });`,
};
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@plane/propel/emoji-icon-picker") return nextResolve(emojiPickerSource, context);
    if (
      context.parentURL?.endsWith("/pages/editor/toolbar/options-dropdown.tsx") &&
      specifier === "../../modals/export-page-modal"
    ) {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export function ExportPageModal() { return null; }")}`,
      };
    }
    if (
      context.parentURL?.endsWith("/pages/editor/toolbar/options-dropdown.tsx") &&
      specifier === "../../navigation-pane"
    ) {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent('export const PAGE_NAVIGATION_PANE_TABS_QUERY_PARAM = "tab";')}`,
      };
    }
    if (context.parentURL?.endsWith("/pages/dropdowns/actions.tsx") && specifier === "../modals/delete-page-modal") {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export function DeletePageModal() { return null; }")}`,
      };
    }
    if (pageMenuStubs[specifier]) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(pageMenuStubs[specifier])}` };
    }
    return nextResolve(specifier, context);
  },
});

let AvatarComponent;
let TooltipComponent;
let PageEditorHeaderRootComponent;

async function renderPageIconPicker(environment, disabled = false) {
  const { PageEditorHeaderRoot } = await import("../../apps/web/core/components/pages/editor/header/root.tsx");
  await environment.render(
    React.createElement(PageEditorHeaderRoot, {
      page: { isContentEditable: !disabled, logo_props: undefined, name: "", updatePageLogo() {} },
    })
  );
}

test("Avatar forwards the Tooltip trigger ref to its rendered element", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  [{ Avatar: AvatarComponent }, { Tooltip: TooltipComponent }] = await Promise.all([
    import("../../packages/ui/src/avatar/avatar.tsx"),
    import("../../packages/propel/src/tooltip/root.tsx"),
  ]);

  const errors = [];
  const originalError = console.error;
  console.error = (...args) => errors.push(args.join(" "));
  t.after(() => {
    console.error = originalError;
  });

  const ref = React.createRef();
  let clickCount = 0;
  await environment.render(
    React.createElement(
      TooltipComponent,
      { tooltipContent: "A tooltip", openDelay: 0 },
      React.createElement(AvatarComponent, {
        name: "Avery",
        ref,
        onClick: () => clickCount++,
        "aria-label": "Avery avatar",
      })
    )
  );

  assert.equal(
    errors.some((message) => message.includes("Function components cannot be given refs")),
    false,
    "Avatar must accept the ref attached by Tooltip.Trigger"
  );
  assert.equal(ref.current?.nodeName.toLowerCase(), "div");
  assert.equal(ref.current.getAttribute("aria-label"), "Avery avatar");
  await React.act(async () => ref.current.dispatchEvent(new MouseEvent("click", { bubbles: true })));
  assert.equal(clickCount, 1, "Avatar preserves caller event handlers on the Tooltip trigger");
});

test("page editor icon picker opens, closes with Escape, and returns focus from search to the trigger", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  await renderPageIconPicker(environment);
  const trigger = environment.container.querySelector("button");
  assert.ok(trigger);
  await React.act(() => trigger.click());
  assert.equal(trigger.getAttribute("aria-expanded"), "true");
  assert.equal(trigger.disabled, false);
  assert.ok(
    document.querySelector('input[placeholder="Search"]'),
    "opening the real icon picker should expose its search field"
  );
  const search = document.querySelector('input[placeholder="Search"]');
  assert.notEqual(document.activeElement, trigger);
  await React.act(() => {
    search.focus();
    search.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
  });
  assert.equal(trigger.getAttribute("aria-expanded"), "false");
  assert.equal(document.activeElement, trigger);
});

test("page editor icon picker has one button trigger", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  ({ PageEditorHeaderRoot: PageEditorHeaderRootComponent } =
    await import("../../apps/web/core/components/pages/editor/header/root.tsx"));
  const errors = [];
  const originalError = console.error;
  console.error = (...args) => errors.push(args.join(" "));
  t.after(() => {
    console.error = originalError;
  });
  const page = {
    isContentEditable: true,
    logo_props: undefined,
    name: "",
    updatePageLogo() {},
  };
  await environment.render(React.createElement(PageEditorHeaderRootComponent, { page }));
  const trigger = environment.container.querySelector("button");
  assert.ok(trigger);
  assert.equal(trigger.querySelector("button"), null, "the label must not nest a button in Popover.Button");
  assert.equal(
    errors.some((message) => message.includes("validateDOMNesting")),
    false
  );
});

test("page editor icon picker keeps a disabled trigger closed", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  await renderPageIconPicker(environment, true);
  const trigger = environment.container.querySelector("button");
  assert.ok(trigger);
  assert.equal(trigger.disabled, true);
  await React.act(() => trigger.click());
  assert.equal(trigger.getAttribute("aria-expanded"), "false");
});

test("page options menu uses one selectable button per toggle item", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  globalThis.__pageActionsTest = {
    fullWidth: false,
    stickyToolbar: true,
    fullWidthChanges: [],
    stickyChanges: [],
  };
  t.after(() => delete globalThis.__pageActionsTest);

  const errors = [];
  const originalError = console.error;
  console.error = (...args) => errors.push(args.join(" "));
  t.after(() => {
    console.error = originalError;
  });

  const { PageOptionsDropdown } =
    await import("../../apps/web/core/components/pages/editor/toolbar/options-dropdown.tsx");
  const page = {
    name: "Lifecycle page",
    isContentEditable: true,
    access: 1,
    archived_at: null,
    is_locked: false,
    canCurrentUserLockPage: false,
    canCurrentUserChangeAccess: false,
    canCurrentUserDuplicatePage: false,
    canCurrentUserArchivePage: false,
    canCurrentUserDeletePage: false,
    canCurrentUserMovePage: false,
    editor: { editorRef: null },
  };
  await environment.render(React.createElement(PageOptionsDropdown, { page, storeType: "project" }));
  const trigger = environment.container.querySelector("button");
  assert.ok(trigger, "the page actions menu has a button trigger");

  await React.act(() => trigger.click());
  const fullWidthItem = [...document.querySelectorAll("button")].find((button) =>
    button.textContent.includes("Full width")
  );
  const stickyItem = [...document.querySelectorAll("button")].find((button) =>
    button.textContent.includes("Sticky toolbar")
  );
  assert.ok(fullWidthItem);
  assert.ok(stickyItem);
  assert.equal(fullWidthItem.querySelector("button"), null);
  assert.equal(stickyItem.querySelector("button"), null);
  assert.equal(fullWidthItem.querySelector('[aria-hidden="true"]')?.nodeName.toLowerCase(), "span");
  assert.equal(fullWidthItem.querySelector(".sr-only")?.textContent, "Off");
  assert.equal(stickyItem.querySelector(".sr-only")?.textContent, "On");

  await React.act(() => fullWidthItem.click());
  assert.deepEqual(globalThis.__pageActionsTest.fullWidthChanges, [true]);
  await React.act(() => trigger.click());
  const reopenedStickyItem = [...document.querySelectorAll("button")].find((button) =>
    button.textContent.includes("Sticky toolbar")
  );
  await React.act(() => reopenedStickyItem.click());
  assert.deepEqual(globalThis.__pageActionsTest.stickyChanges, [false]);
  assert.equal(
    errors.some((message) => message.includes("validateDOMNesting")),
    false
  );
});
