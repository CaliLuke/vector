import { changeFormControl } from "./helpers/native-forms.mjs";
import { settleQuery } from "./helpers/query-router.mjs";
import { seedInstanceInfo } from "./helpers/instance-info.mjs";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createRouterFixture } from "./helpers/router.mjs";
import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";

registerWebModuleHooks();

globalThis.imagePickerFileServiceFixture = {
  searches: [],
  FileService: class FileService {
    async getUnsplashImages(query) {
      globalThis.imagePickerFileServiceFixture.searches.push(query);
      return [];
    }
  },
};
const modules = new Map(
  Object.entries({
    "@/helpers/cover-image.helper": `export const STATIC_COVER_IMAGES = []; export const getCoverImageDisplayURL = value => value;`,
    "@/services/file.service": `export const searches=globalThis.imagePickerFileServiceFixture.searches;export const FileService=globalThis.imagePickerFileServiceFixture.FileService;`,
  }).map(([name, source]) => [name, `data:text/javascript,${encodeURIComponent(source)}`])
);

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (modules.has(specifier)) return { shortCircuit: true, url: modules.get(specifier) };
    return nextResolve(specifier, context);
  },
});

test("cover image search retains typed text and searches without a parent form control", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const routerFixture = createRouterFixture("/review/");
  t.after(() => environment.unmount());
  const { ImagePickerPopover } = await import("../../apps/web/core/components/core/image-picker-popover.tsx");
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  t.after(() => queryClient.clear());
  seedInstanceInfo(queryClient, { has_unsplash_configured: true });
  await routerFixture.render(
    environment,
    React.createElement(
      QueryClientProvider,
      { client: queryClient },
      React.createElement(ImagePickerPopover, {
        label: "Choose cover",
        workspaceSlug: "review",
        value: null,
        onChange: () => assert.fail("searching must not change the selected cover"),
      })
    )
  );
  await React.act(async () => environment.container.querySelector("button").click());
  const input = environment.container.querySelector('input[placeholder="Search for images"]');
  assert.ok(input, "the image-search input should be available");
  const setValue = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, "value").set;
  await React.act(async () => {
    setValue.call(input, "mountains");
    input.dispatchEvent(new window.Event("input", { bubbles: true }));
  });
  assert.equal(input.value, "mountains");
  const search = [...environment.container.querySelectorAll("button")].find(
    (button) => button.textContent === "Search"
  );
  assert.ok(search);
  await React.act(async () => search.click());
  assert.equal(globalThis.imagePickerFileServiceFixture.searches.at(-1), "mountains");
  assert.equal(input.value, "mountains");
});

await test("failed cover searches keep the input available and retry the same query", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const routerFixture = createRouterFixture("/review/");
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  t.after(async () => {
    await environment.unmount();
    client.clear();
  });
  const { ImagePickerPopover } = await import("../../apps/web/core/components/core/image-picker-popover.tsx");
  const { FileService } = globalThis.imagePickerFileServiceFixture;
  const fetch = t.mock.method(FileService.prototype, "getUnsplashImages", async () => {
    throw new Error("offline");
  });
  seedInstanceInfo(client, { has_unsplash_configured: true });
  await routerFixture.render(
    environment,
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(ImagePickerPopover, {
        label: "Choose cover",
        workspaceSlug: "review",
        value: null,
        onChange: () => {},
      })
    )
  );
  await React.act(async () => environment.container.querySelector("button").click());
  await settleQuery();
  const input = environment.container.querySelector('input[placeholder="Search for images"]');
  assert.ok(input);
  await changeFormControl(input, "mountains");
  const button = (label) =>
    [...environment.container.querySelectorAll("button")].find((node) => node.textContent === label);
  await React.act(async () => button("Search").click());
  await settleQuery();
  assert.match(environment.container.querySelector('[role="alert"]').textContent, /could not load/i);
  assert.equal(input.value, "mountains");
  fetch.mock.mockImplementation(async () => []);
  await React.act(async () => button("Retry search").click());
  await settleQuery();
  assert.equal(fetch.mock.calls.at(-1).arguments[0], "mountains");
  assert.equal(environment.container.querySelector('[role="alert"]'), null);
  assert.equal(input.value, "mountains");
});
