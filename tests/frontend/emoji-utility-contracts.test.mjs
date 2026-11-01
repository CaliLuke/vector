import assert from "node:assert/strict";
import test from "node:test";
import { convertHexEmojiToDecimal, emojiCodeToUnicode, groupReactions } from "../../packages/utils/src/emoji.ts";

await test("reaction grouping preserves row identity and safely accepts prototype-named groups", (t) => {
  const warning = t.mock.method(console, "warn", () => {});
  const ordinary = { code: "128512", user: "alice" };
  const duplicate = { code: "128512", user: "bob" };
  const special = { code: "__proto__", user: "carol" };
  const constructor = { code: "constructor", user: "dave" };
  const inherited = Object.create({ code: "inherited" });
  const grouped = groupReactions([ordinary, null, {}, inherited, special, constructor, duplicate], "code");
  assert.deepEqual(Object.keys(grouped).toSorted(), ["128512", "__proto__", "constructor"]);
  assert.deepEqual(grouped["128512"], [ordinary, duplicate]);
  assert.equal(grouped["128512"][0], ordinary);
  assert.equal(grouped.__proto__[0], special);
  assert.equal(grouped.constructor[0], constructor);
  assert.equal(Object.getPrototypeOf(grouped), Object.prototype);
  assert.equal(warning.mock.callCount(), 3);
});

await test("emoji conversion retains multi-codepoint and empty values", () => {
  assert.equal(convertHexEmojiToDecimal("1f1e6-1f1e8"), "127462-127464");
  assert.equal(emojiCodeToUnicode("127462-127464"), "1f1e6-1f1e8");
  assert.equal(convertHexEmojiToDecimal(""), "");
  assert.equal(emojiCodeToUnicode(""), "");
});
