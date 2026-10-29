import { defineConfig } from "tsdown";

export default defineConfig({
  entry: [
    "src/index.ts",
    "src/button/index.ts",
    "src/dropdowns/index.ts",
    "src/form-fields/index.ts",
    "src/loader.tsx",
    "src/modals/index.ts",
  ],
  format: ["esm"],
  dts: true,
  platform: "neutral",
  exports: true,
});
