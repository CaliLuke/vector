import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { itemAssetReferences } =
  await import("../../apps/web/core/components/issues/issue-modal/item-asset-references.ts");

await test("only visible rich text and description images finalize uploads", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const schema = {
    properties: [
      { id: "rich", kind: "Rich text" },
      { id: "text", kind: "Text" },
    ],
  };
  const values = {
    description_html: '<p>Text</p><image-component src="description"></image-component><img src="shared">',
    custom_properties: {
      rich: '<img src="shared"><image-component src="rich-image"></image-component><iframe src="non-image"></iframe>',
      text: '<img src="literal-markup">',
      hidden: '<img src="other-type">',
    },
  };
  assert.deepEqual(itemAssetReferences(values, schema), ["description", "shared", "rich-image"]);
  values.custom_properties.rich = "<p>Image removed</p>";
  assert.deepEqual(itemAssetReferences(values, schema), ["description", "shared"]);
});
