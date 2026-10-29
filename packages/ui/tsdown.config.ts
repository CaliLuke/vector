import { defineConfig } from "tsdown";

export default defineConfig({
  entry: [
    "src/index.ts",
    "src/avatar/index.ts",
    "src/button/index.ts",
    "src/content-wrapper/index.ts",
    "src/control-link/index.ts",
    "src/dropdowns/index.ts",
    "src/form-fields/index.ts",
    "src/header/index.ts",
    "src/loader.tsx",
    "src/modals/index.ts",
    "src/popovers/index.ts",
    "src/row/index.ts",
    "src/tooltip/index.ts",
  ],
  format: ["esm"],
  dts: true,
  platform: "neutral",
  exports: true,
});
