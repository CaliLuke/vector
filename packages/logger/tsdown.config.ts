import { defineConfig } from "tsdown";

export default defineConfig({
  sourcemap: true,
  entry: ["src/index.ts"],
  format: ["esm"],
  deps: { neverBundle: ["express"] },
  dts: true,
  exports: true,
});
