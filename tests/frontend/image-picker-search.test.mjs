import { createRouterFixture } from "./helpers/router.mjs";
import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createReactDomTestEnvironment, React } from "./helpers/react-dom.mjs";

registerWebModuleHooks();

const modules = new Map(
  Object.entries({
    "@/hooks/store/use-instance": `export const useInstance = () => ({ config: { has_unsplash_configured: true } });`,
    "@/helpers/cover-image.helper": `export const STATIC_COVER_IMAGES = []; export const getCoverImageDisplayURL = value => value;`,
    "@/services/file.service": `
    export const searches = [];
    export class FileService {
      async getUnsplashImages(query) { searches.push(query); return []; }
    }
  `,
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
  const { searches } = await import("@/services/file.service");
  await routerFixture.render(
    environment,
    React.createElement(ImagePickerPopover, {
      label: "Choose cover",
      value: null,
      onChange: () => assert.fail("searching must not change the selected cover"),
    })
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
  assert.equal(searches.at(-1), "mountains");
  assert.equal(input.value, "mountains");
});
