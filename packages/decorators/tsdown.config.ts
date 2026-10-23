import { defineConfig } from "tsdown";

export default defineConfig({
  entry: ["src/index.ts"],
  format: ["esm"],
  external: ["express"],
  dts: true,
  exports: true,
});
