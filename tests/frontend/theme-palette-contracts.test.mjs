import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { calculateDynamicValueStop, generateColorPalette } =
  await import("../../packages/utils/src/theme/palette-generator.ts");
const { hexToOKLCH, oklchToCSS, parseOKLCH } = await import("../../packages/utils/src/theme/color-conversion.ts");
const { normalizeHexColor, validateHexColor } = await import("../../packages/utils/src/theme/color-validation.ts");
const { invertPalette, getNeutralMapping, getBrandMapping } =
  await import("../../packages/utils/src/theme/theme-inversion.ts");

void test("automatic palette anchoring keeps the input color at its closest lightness stop", () => {
  const input = hexToOKLCH("#3468db");
  const anchor = calculateDynamicValueStop(input);
  const palette = generateColorPalette("3468db", "light", { valueStop: "auto" });
  assert.equal(palette[anchor], oklchToCSS(input));
  assert.equal(Object.keys(palette).length, 14);
  assert.equal(calculateDynamicValueStop({ l: 1, c: 0, h: 0 }), 50);
  assert.equal(calculateDynamicValueStop({ l: 0, c: 0, h: 0 }), 1000);
});

void test("palette generation rejects invalid colors and unsupported anchor stops", () => {
  assert.throws(() => generateColorPalette("not-a-color", "light"), /Invalid hex color/);
  assert.throws(() => generateColorPalette("#3468db", "dark", { valueStop: 123 }), /Invalid valueStop/);
});

void test("color inputs normalize shorthand, reject empty values, and keep extreme colors visible", () => {
  assert.equal(normalizeHexColor("#a0f"), "AA00FF");
  assert.equal(validateHexColor(""), false);
  assert.equal(hexToOKLCH("#fff").l, 0.9);
  assert.equal(hexToOKLCH("#000").l, 0.15);
  assert.equal(hexToOKLCH("#000").c, 0.002);
});

void test("OKLCH parsing preserves valid channels and rejects unrelated CSS", () => {
  assert.deepEqual(parseOKLCH("oklch(0.5840 0.1200 250.00)"), { l: 0.584, c: 0.12, h: 250 });
  assert.equal(parseOKLCH("#3468db"), null);
  assert.equal(oklchToCSS({ l: 0.5, c: 0.1, h: 250 }, 50), "oklch(0.5000 0.1000 250.00 / 50.00%)");
});

void test("invalid conversion input returns the safe fallback color", (t) => {
  t.mock.method(console, "error", () => {});
  assert.deepEqual(hexToOKLCH("invalid"), { l: 0.5, c: 0, h: 0 });
});

void test("theme mappings retain shared shades, neutral endpoints and the brand anchor", () => {
  const palette = generateColorPalette("#3468db", "light");
  const neutral = getNeutralMapping(palette);
  const brand = getBrandMapping(palette);
  assert.equal(neutral.white, palette[50]);
  assert.equal(neutral.black, palette[1000]);
  assert.equal(brand.default, palette[500]);
  assert.equal(brand[800], palette[750]);
  assert.equal(brand[800], neutral[800]);
  assert.equal(brand[1200], palette[950]);
  assert.equal(brand[1200], neutral[1200]);
  assert.deepEqual(invertPalette(invertPalette(palette)), palette);
});
