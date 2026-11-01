import { defineConfig } from "tsdown";

export default defineConfig({
  sourcemap: true,
  entry: ["src/index.ts"],
  format: ["esm"],
  dts: true,
  exports: true,
});
