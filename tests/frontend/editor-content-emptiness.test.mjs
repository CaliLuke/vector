import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { isJSONContentEmpty } = await import("../../packages/utils/src/string.ts");

void test("editor emptiness checks safely handle malformed imported content", () => {
  for (const content of [
    null,
    undefined,
    [],
    42,
    "text",
    {},
    { text: {} },
    { content: {} },
    { type: "doc", content: [null, { type: "paragraph" }, { text: "  " }] },
  ]) {
    assert.equal(isJSONContentEmpty(content), true);
  }
  assert.equal(
    isJSONContentEmpty({ type: "doc", content: [{ type: "paragraph", content: [{ type: "text", text: "Body" }] }] }),
    false
  );
});

void test("image-only and mention-only documents retain current and legacy editor nodes", () => {
  for (const type of ["hardBreak", "image", "imageComponent", "image-component", "mention", "mention-component"]) {
    assert.equal(
      isJSONContentEmpty({ type: "doc", content: [{ type, attrs: { src: "asset", entity_identifier: "user" } }] }),
      false,
      type
    );
  }
});
