import { describe, expect, it } from "vitest";
import { resolveColorForPdf } from "@/lib/pdf/colors";
import { applyMarks } from "@/lib/pdf/mark-renderers";

describe("PDF editor color resolution", () => {
  it.each<[string, "text" | "background", string]>([
    ["green", "text", "#0fc27b"],
    ["green", "background", "#c3f0de"],
    ["var(--editor-colors-light-blue-text)", "text", "#17bee9"],
    ["var(--editor-colors-light-blue-background)", "background", "#c5eff9"],
    ["#123abc", "text", "#123abc"],
    ["#abc", "background", "#abc"],
  ])("resolves %s for %s", (value, type, expected) => {
    expect(resolveColorForPdf(value, type)).toBe(expected);
  });

  it.each([
    undefined,
    null,
    "",
    "unknown",
    "var(--other-color)",
    "var(--editor-colors-unknown-text)",
    "__proto__",
    "constructor",
    "toString",
  ])("ignores unsupported color %s", (value) => {
    expect(resolveColorForPdf(value, "text")).toBeNull();
    expect(resolveColorForPdf(value, "background")).toBeNull();
  });
});

describe("PDF custom color marks", () => {
  it("combines text and background colors without changing the supplied style", () => {
    const base = { fontSize: 14, color: "#000000" };
    expect(
      applyMarks(
        [{ type: "customColor", attrs: { color: "var(--editor-colors-light-blue-text)", backgroundColor: "green" } }],
        base
      )
    ).toEqual({
      fontSize: 14,
      color: "#17bee9",
      backgroundColor: "#c3f0de",
    });
    expect(base).toEqual({ fontSize: 14, color: "#000000" });
  });

  it("preserves direct RGB colors", () => {
    expect(
      applyMarks([
        { type: "customColor", attrs: { color: "rgb(10, 20, 30)", backgroundColor: "rgba(40, 50, 60, 0.5)" } },
      ])
    ).toEqual({
      color: "rgb(10, 20, 30)",
      backgroundColor: "rgba(40, 50, 60, 0.5)",
    });
  });

  it.each(["unknown", "__proto__", "constructor", "toString", 42, { invalid: true }])(
    "ignores invalid custom color attributes %j",
    (color) => {
      const base = { color: "#123456", backgroundColor: "#abcdef" };
      expect(applyMarks([{ type: "customColor", attrs: { color, backgroundColor: color } }], base)).toEqual(base);
    }
  );

  it.each(["unknown", "__proto__", "constructor", "toString"])("ignores an unknown mark named %s", (type) => {
    const base = { fontSize: 14 };
    expect(applyMarks([{ type }, { type: "bold" }], base)).toEqual({ fontSize: 14, fontWeight: "bold" });
  });
});
