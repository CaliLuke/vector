import { defineConfig } from "tsdown";

export default defineConfig({
  entry: ["src/index.ts"],
  format: ["esm"],
  deps: { neverBundle: ["express"] },
  dts: true,
  exports: true,
});
