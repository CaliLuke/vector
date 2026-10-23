import { createRouterFixture } from "./helpers/router.mjs";
import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import path from "node:path";
import test from "node:test";
import { fileURLToPath, pathToFileURL } from "node:url";
import { React, createReactDomTestEnvironment, webRequire } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const repositoryRoot = fileURLToPath(new URL("../..", import.meta.url));
const reactModuleUrl = pathToFileURL(webRequire.resolve("react")).href;
const { useForm } = webRequire("react-hook-form");
const packageSources = {
  "@plane/propel/button": "packages/propel/src/button/index.ts",
  "@plane/propel/tabs": "packages/propel/src/tabs/index.ts",
};
const packageStubs = {
  "@plane/constants":
    "export const ACCEPTED_COVER_IMAGE_MIME_TYPES_FOR_REACT_DROPZONE = { 'image/*': ['.jpg', '.jpeg', '.png', '.webp'] }; export const MAX_FILE_SIZE = 5242880;",
  "@plane/propel/toast": "export const TOAST_TYPE = { ERROR: 'error' }; export const setToast = () => {};",
  "@plane/types": "export const EFileAssetType = { USER_COVER: 'user-cover', PROJECT_COVER: 'project-cover' };",
  "@plane/ui": `import React from ${JSON.stringify(reactModuleUrl)};
    export const Input = React.forwardRef((props, ref) => React.createElement("input", { ...props, ref }));
    const Loader = ({ children, ...props }) => React.createElement("div", props, children);
    Loader.Item = (props) => React.createElement("div", props);
    export { Loader };`,
  "@/helpers/cover-image.helper": `export const STATIC_COVER_IMAGES = { IMAGE_1: "/cover-one.jpg", IMAGE_2: "/cover-two.jpg" };
    export const getCoverImageDisplayURL = (value, fallback) => value || fallback;`,
  "@/hooks/store/use-instance": "export const useInstance = () => ({ config: { has_unsplash_configured: false } });",
  "@/services/file.service": `export class FileService {
    getUnsplashImages() { return Promise.resolve([]); }
    uploadUserAsset() { return Promise.reject(new Error("upload not configured in this test")); }
    uploadWorkspaceAsset() { return Promise.reject(new Error("upload not configured in this test")); }
  }`,
  "mobx-react": "export const observer = (component) => component;",

  "react-dropzone": `export const useDropzone = () => ({
    getRootProps: () => ({}),
    getInputProps: () => ({ type: "file" }),
    isDragActive: false,
    fileRejections: [],
  });`,
  swr: "export default () => ({ data: [], error: null });",
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

let imagePickerPopoverPromise;
let ImagePickerComponent;
async function loadImagePicker() {
  imagePickerPopoverPromise ??= import("../../apps/web/core/components/core/image-picker-popover.tsx").then(
    (module) => module.ImagePickerPopover
  );
  return imagePickerPopoverPromise;
}

function PickerHarness({ disabled = false, value = null, onChange = () => {} } = {}) {
  const { control } = useForm({ defaultValues: { search: "" } });
  return React.createElement(ImagePickerComponent, {
    label: "Change cover",
    value,
    control,
    onChange,
    disabled,
    isProfileCover: true,
  });
}

async function renderPicker(environment, props) {
  const routerFixture = createRouterFixture("/workspace/");
  ImagePickerComponent = await loadImagePicker();
  await routerFixture.render(environment, React.createElement(PickerHarness, props));
  return environment.container.querySelector("button");
}

function keyDown(element, key) {
  element.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true }));
}

test("cover picker uses native Popover state for mouse and keyboard activation", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const trigger = await renderPicker(environment);

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
