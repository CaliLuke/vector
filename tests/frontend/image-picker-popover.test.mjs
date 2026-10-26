import { settleQuery } from "./helpers/query-router.mjs";
import { seedInstanceInfo } from "./helpers/instance-info.mjs";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createRouterFixture } from "./helpers/router.mjs";
import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import path from "node:path";
import test, { beforeEach } from "node:test";
import { fileURLToPath, pathToFileURL } from "node:url";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, webRequire } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const repositoryRoot = fileURLToPath(new URL("../..", import.meta.url));
const reactModuleUrl = pathToFileURL(webRequire.resolve("react")).href;
const packageSources = {
  "@plane/propel/button": "packages/propel/src/button/index.ts",
  "@plane/propel/tabs": "packages/propel/src/tabs/index.ts",
};
const packageStubs = {
  "@plane/propel/toast": "export const TOAST_TYPE = { ERROR: 'error' }; export const setToast = () => {};",
  "@plane/ui": `import React from ${JSON.stringify(reactModuleUrl)};
    export const Input = React.forwardRef((props, ref) => React.createElement("input", { ...props, ref }));
    const Loader = ({ children, ...props }) => React.createElement("div", props, children);
    Loader.Item = (props) => React.createElement("div", props);
    export { Loader };`,
  "@/helpers/cover-image.helper": `export const STATIC_COVER_IMAGES = { IMAGE_1: "/cover-one.jpg", IMAGE_2: "/cover-two.jpg" };
    export const getCoverImageDisplayURL = (value, fallback) => value || fallback;`,
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (packageSources[specifier]) {
      return nextResolve(pathToFileURL(path.join(repositoryRoot, packageSources[specifier])).href, context);
    }
    if (Object.hasOwn(packageStubs, specifier)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(packageStubs[specifier])}` };
    }
    return nextResolve(specifier, context);
  },
});

const { FileService } = await import("../../apps/web/core/services/file.service.ts");
beforeEach((t) => {
  t.mock.method(FileService.prototype, "getUnsplashImages", async () => []);
});

let imagePickerPopoverPromise;
let ImagePickerComponent;
async function loadImagePicker() {
  imagePickerPopoverPromise ??= import("../../apps/web/core/components/core/image-picker-popover.tsx").then(
    (module) => module.ImagePickerPopover
  );
  return imagePickerPopoverPromise;
}

function PickerHarness({ disabled = false, value = null, onChange = () => {} } = {}) {
  const [queryClient] = React.useState(() => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
    seedInstanceInfo(client, { has_unsplash_configured: false });
    return client;
  });
  React.useEffect(() => () => queryClient.clear(), [queryClient]);
  return React.createElement(
    QueryClientProvider,
    { client: queryClient },
    React.createElement(ImagePickerComponent, {
      label: "Change cover",
      value,
      onChange,
      disabled,
      isProfileCover: true,
    })
  );
}

async function renderPicker(environment, props) {
  const routerFixture = createRouterFixture("/settings/profile/general/");
  ImagePickerComponent = await loadImagePicker();
  await routerFixture.render(environment, React.createElement(PickerHarness, props));
  return environment.container.querySelector("button");
}

function keyDown(element, key) {
  element.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true }));
}

test("profile cover picker opens on the global settings route with mouse and keyboard activation", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const trigger = await renderPicker(environment);
  assert.equal(
    FileService.prototype.getUnsplashImages.mock.callCount(),
    0,
    "unconfigured Unsplash does not send a request"
  );

  await React.act(() => trigger.click());
  assert.equal(trigger.getAttribute("aria-expanded"), "true");
  assert.ok(document.querySelector('[role="tablist"]'));

  await React.act(() => trigger.click());
  assert.equal(trigger.getAttribute("aria-expanded"), "false");
  assert.equal(document.querySelector('[role="tablist"]'), null);

  await React.act(async () => {
    trigger.focus();
    keyDown(trigger, "Enter");
  });
  assert.equal(trigger.getAttribute("aria-expanded"), "true");
  assert.ok(document.querySelector('[role="tablist"]'));
});

test("cover picker reports open state and closes with Escape while returning focus", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const trigger = await renderPicker(environment);

  await React.act(() => trigger.click());
  const uploadTab = [...document.querySelectorAll('[role="tab"]')].find((tab) => tab.textContent === "Upload");
  assert.ok(uploadTab);
  await React.act(() => uploadTab.click());
  assert.equal(uploadTab.getAttribute("aria-selected"), "true");
  assert.ok(document.querySelector('input[type="file"]'));

  const fileInput = document.querySelector('input[type="file"]');
  await React.act(() => {
    fileInput.focus();
    keyDown(fileInput, "Escape");
  });
  assert.equal(trigger.getAttribute("aria-expanded"), "false");
  assert.equal(document.querySelector('[role="tablist"]'), null);
  assert.equal(document.activeElement, trigger);
});

test("Enter inside the Upload tab does not toggle the cover picker", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const trigger = await renderPicker(environment);

  await React.act(() => trigger.click());
  const uploadTab = [...document.querySelectorAll('[role="tab"]')].find((tab) => tab.textContent === "Upload");
  await React.act(() => uploadTab.click());
  const fileInput = document.querySelector('input[type="file"]');

  await React.act(() => {
    fileInput.focus();
    keyDown(fileInput, "Enter");
  });
  assert.equal(trigger.getAttribute("aria-expanded"), "true");
  assert.ok(document.querySelector('[role="tablist"]'));
});

test("cover picker keeps upload and save disabled without a file and honors a disabled trigger", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const trigger = await renderPicker(environment);

  await React.act(() => trigger.click());
  const uploadTab = [...document.querySelectorAll('[role="tab"]')].find((tab) => tab.textContent === "Upload");
  await React.act(() => uploadTab.click());
  const save = [...document.querySelectorAll("button")].find((button) => button.textContent.includes("Upload & Save"));
  assert.ok(save);
  assert.equal(save.disabled, true);

  await environment.unmount();
  const disabledEnvironment = await createReactDomTestEnvironment();
  t.after(() => disabledEnvironment.unmount());
  const disabledTrigger = await renderPicker(disabledEnvironment, { disabled: true });
  assert.equal(disabledTrigger.disabled, true);
  await React.act(() => disabledTrigger.click());
  assert.equal(disabledTrigger.getAttribute("aria-expanded"), "false");
  assert.equal(document.querySelector('[role="tablist"]'), null);
});

test("Cancel and static image selection close the popover, and selection updates the cover", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const selected = [];
  const trigger = await renderPicker(environment, { onChange: (value) => selected.push(value) });

  await React.act(() => trigger.click());
  const uploadTab = [...document.querySelectorAll('[role="tab"]')].find((tab) => tab.textContent === "Upload");
  await React.act(() => uploadTab.click());
  const cancel = [...document.querySelectorAll("button")].find((button) => button.textContent === "Cancel");
  await React.act(() => cancel.click());
  assert.equal(trigger.getAttribute("aria-expanded"), "false");
  assert.equal(document.querySelector('[role="tablist"]'), null);

  await React.act(() => trigger.click());
  const image = document.querySelector('img[alt="Cover 1"]');
  assert.ok(image);
  assert.equal(image.closest("button").type, "button");
  await React.act(() => image.click());
  assert.equal(trigger.getAttribute("aria-expanded"), "false");
  assert.deepEqual(selected, ["/cover-one.jpg"]);
});

await test("static selection waits for its owner and leaves a failed selection open for retry", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const pending = Promise.withResolvers();
  const owner = t.mock.fn(async () => {});
  owner.mock.mockImplementationOnce(() => pending.promise);
  const trigger = await renderPicker(environment, { onChange: owner });
  await React.act(async () => trigger.click());
  await React.act(async () => document.querySelector('img[alt="Cover 1"]').click());
  assert.equal(trigger.getAttribute("aria-expanded"), "true");
  await React.act(async () => {
    pending.reject(new Error("save failed"));
    await Promise.resolve();
  });
  await settleQuery();
  assert.match(document.querySelector('[role="alert"]').textContent, /try again/i);
  assert.equal(trigger.getAttribute("aria-expanded"), "true");
  await React.act(async () => document.querySelector('img[alt="Cover 1"]').click());
  assert.equal(owner.mock.callCount(), 2);
  assert.equal(trigger.getAttribute("aria-expanded"), "false");
});

await test("uploaded cover retries the owner callback without duplicating the asset", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  t.mock.method(URL, "createObjectURL", () => "blob:cover-preview");
  const revoke = t.mock.method(URL, "revokeObjectURL", () => {});
  const upload = t.mock.method(FileService.prototype, "uploadUserAsset", async () => ({ asset_url: "/asset/cover/" }));
  const owner = t.mock.fn(async () => {});
  owner.mock.mockImplementationOnce(async () => {
    throw new Error("save failed");
  });
  const trigger = await renderPicker(environment, { onChange: owner });
  await React.act(async () => trigger.click());
  await React.act(async () =>
    [...document.querySelectorAll('[role="tab"]')].find((tab) => tab.textContent === "Upload").click()
  );
  const file = new File(["cover"], "cover.png", { type: "image/png" });
  await React.act(async () => {
    const input = document.querySelector('input[type="file"]');
    Object.defineProperty(input, "files", { configurable: true, value: [file] });
    input.dispatchEvent(new Event("change", { bubbles: true }));
    await Promise.resolve();
  });
  const save = () =>
    [...environment.container.querySelectorAll("button")].find((button) => button.textContent === "Upload & Save");
  await React.act(async () => save().click());
  assert.equal(trigger.getAttribute("aria-expanded"), "true");
  assert.equal(upload.mock.callCount(), 1);
  await React.act(async () => save().click());
  assert.equal(upload.mock.callCount(), 1);
  assert.equal(owner.mock.callCount(), 2);
  assert.equal(trigger.getAttribute("aria-expanded"), "false");
  assert.equal(revoke.mock.calls[0].arguments[0], "blob:cover-preview");
});
