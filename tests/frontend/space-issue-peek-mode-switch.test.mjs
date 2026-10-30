import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import assert from "node:assert/strict";
import { createRequire, registerHooks } from "node:module";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerSpaceModuleHooks } from "./helpers/web-module-hooks.mjs";

const { MemoryRouter, useSearchParams } = createRequire(new URL("../../apps/space/package.json", import.meta.url))(
  "react-router"
);

registerSpaceModuleHooks();

const repositoryRoot = fileURLToPath(new URL("../../", import.meta.url));
const packageStubs = {
  "@plane/propel/icons": `export const MoveRight = () => null;
    export const LinkIcon = () => null;
    export const CenterPanelIcon = () => null;
    export const FullScreenPanelIcon = () => null;
    export const SidePanelIcon = () => null;`,
  "@plane/propel/toast": "export const TOAST_TYPE = { SUCCESS: 'success' }; export const setToast = () => {};",
  "@plane/ui": `export const Loader = Object.assign(() => null, { Item: () => null });`,
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@/helpers/string.helper") {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export const copyTextToClipboard = async () => {};")}`,
      };
    }
    if (specifier === "./issue-activity" || specifier === "./issue-details" || specifier === "./issue-properties") {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export const PeekOverviewIssueActivity = () => null; export const PeekOverviewIssueDetails = () => null; export const PeekOverviewIssueProperties = () => null;")}`,
      };
    }
    if (Object.hasOwn(packageStubs, specifier)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(packageStubs[specifier])}` };
    }
    if (specifier === "@headlessui/react") {
      const webRoot = path.join(repositoryRoot, "apps/web");
      return nextResolve(specifier, { ...context, parentURL: pathToFileURL(path.join(webRoot, "index.js")).href });
    }
    return nextResolve(specifier, context);
  },
});

let IssuePeekOverview;

function TestHarness() {
  const [params, setParams] = useSearchParams();
  return React.createElement(
    React.Fragment,
    null,
    React.createElement("button", { id: "open-peek", onClick: () => setParams({ peekId: "issue-1" }) }, "Open issue"),
    React.createElement(IssuePeekOverview, { anchor: "project", peekId: params.get("peekId") ?? "" })
  );
}

function getDialog() {
  return document.querySelector('[role="dialog"]');
}
async function waitForDialog(expected) {
  const deadline = Date.now() + 1000;
  while (Boolean(getDialog()) !== expected && Date.now() < deadline) {
    // Wait for the real dialog transition before checking its final DOM state.
    // eslint-disable-next-line no-await-in-loop
    await React.act(async () => new Promise((resolve) => setTimeout(resolve, 20)));
  }
  assert.equal(Boolean(getDialog()), expected);
}
async function flushAnimationFrames() {
  await new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve)));
}

test("switching peek modes keeps one dialog, focus, Escape close, and full-screen sizing", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(async () => {
    await environment.unmount();
    delete globalThis.ResizeObserver;
  });
  globalThis.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  };

  const warnings = [];
  const originalWarn = console.warn;
  console.warn = (...args) => warnings.push(args.join(" "));
  t.after(() => {
    console.warn = originalWarn;
  });

  ({ IssuePeekOverview } = await import("../../apps/space/components/issues/peek-overview/layout.tsx"));
  const client = new QueryClient();
  const { issueOptions } = await import("../../apps/space/queries/issues.ts");
  const { publishSettingsOptions } = await import("../../apps/space/queries/publishing.ts");
  client.setQueryData(issueOptions("project", "issue-1").queryKey, { id: "issue-1" });
  client.setQueryData(publishSettingsOptions("project").queryKey, { is_comments_enabled: false });
  t.after(() => client.clear());
  await environment.render(
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(MemoryRouter, null, React.createElement(TestHarness))
    )
  );
  const opener = environment.container.querySelector("#open-peek");
  opener.focus();
  await React.act(async () => {
    opener.click();
    await flushAnimationFrames();
  });
  const hasClassPart = (part) =>
    [...(getDialog()?.querySelectorAll("[class]") ?? [])].some((element) => element.className.includes(part));
  const modeTrigger = () => getDialog()?.querySelector('button[aria-haspopup="listbox"]');
  assert.ok(getDialog(), "the issue peek should open in a dialog");
  assert.ok(hasClassPart("w-1/2"));

  async function chooseMode(label) {
    const trigger = modeTrigger();
    assert.ok(trigger, "the mode selector should remain mounted");
    await React.act(async () => {
      trigger.click();
      await flushAnimationFrames();
    });
    const option = [...document.querySelectorAll('[role="option"]')].find((item) => item.textContent.includes(label));
    assert.ok(option, `the ${label} option should be available`);
    await React.act(async () => {
      option.click();
      await flushAnimationFrames();
    });
    assert.equal(document.querySelectorAll('[role="dialog"]').length, 1, "mode changes should retain one dialog");
    assert.ok(
      getDialog().contains(document.activeElement),
      `focus should remain inside the active dialog (active=${document.activeElement?.outerHTML ?? "none"})`
    );
  }

  await chooseMode("Modal");
  assert.ok(hasClassPart("h-[70%]"), "modal dimensions should remain unchanged");
  await chooseMode("Full Screen");
  assert.ok(hasClassPart("size-[95%]"), "full-screen dimensions should remain unchanged");
  await chooseMode("Side Peek");
  assert.ok(hasClassPart("w-1/2"), "side-peek dimensions should remain unchanged");
  assert.equal(
    warnings.some((warning) => warning.includes("There are no focusable elements inside the <FocusTrap />")),
    false
  );

  const activeDialog = getDialog();
  await React.act(() => {
    activeDialog.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
  });
  await waitForDialog(false);
  assert.equal(getDialog(), null, "Escape should close the peek");
  assert.equal(document.activeElement?.id, "open-peek", "closing the peek should restore focus to its opener");
});
