import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import test from "node:test";
import { React, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerSpaceModuleHooks } from "./helpers/web-module-hooks.mjs";

registerSpaceModuleHooks();

const repositoryRoot = fileURLToPath(new URL("../../", import.meta.url));
const stateStub = `data:text/javascript,${encodeURIComponent(`
  const state = globalThis.__spacePeekTestState ??= { mode: "side", open: true };
  const notify = () => window.dispatchEvent(new Event("space-peek-state-change"));
  export const useIssueDetails = () => ({
    get peekMode() { return state.mode; },
    setPeekMode(mode) { state.mode = mode; notify(); },
    setPeekId(value) { state.open = value !== null; notify(); },
    getIssueById() { return { id: "issue-1" }; },
    fetchIssueDetails() {},
  });
`)}`;

const packageStubs = {
  "@plane/propel/icons": `export const MoveRight = () => null;
    export const LinkIcon = () => null;
    export const CenterPanelIcon = () => null;
    export const FullScreenPanelIcon = () => null;
    export const SidePanelIcon = () => null;`,
  "@plane/propel/toast": "export const TOAST_TYPE = { SUCCESS: 'success' }; export const setToast = () => {};",
  "@plane/ui": `export const Loader = Object.assign(() => null, { Item: () => null });`,
  "mobx-react": "export const observer = component => component;",
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@/hooks/store/use-issue-details") return { shortCircuit: true, url: stateStub };
    if (specifier === "next/navigation") {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent(`
          export const useRouter = () => ({ push() {} });
          export const useSearchParams = () => new URLSearchParams();
        `)}`,
      };
    }
    if (specifier === "@/hooks/store/publish") {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export const usePublish = () => ({ canComment: false });")}`,
      };
    }
    if (specifier === "@/hooks/use-clipboard-write-permission") {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent("export default () => false;")}` };
    }
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
  const [, setRevision] = React.useState(0);
  React.useEffect(() => {
    const update = () => setRevision((revision) => revision + 1);
    window.addEventListener("space-peek-state-change", update);
    return () => window.removeEventListener("space-peek-state-change", update);
  }, []);

  return React.createElement(
    React.Fragment,
    null,
    React.createElement(
      "button",
      {
        type: "button",
        id: "open-peek",
        onClick() {
          globalThis.__spacePeekTestState.open = true;
          window.dispatchEvent(new Event("space-peek-state-change"));
        },
      },
      "Open issue"
    ),
    React.createElement(IssuePeekOverview, {
      anchor: "project",
      peekId: globalThis.__spacePeekTestState.open ? "issue-1" : "",
    })
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
  globalThis.__spacePeekTestState = { mode: "side", open: false };
  const environment = await createReactDomTestEnvironment();
  t.after(async () => {
    await environment.unmount();
    delete globalThis.__spacePeekTestState;
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
  await environment.render(React.createElement(TestHarness));
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
