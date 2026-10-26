import path from "node:path";
import * as dotenv from "dotenv";
import { reactRouter } from "@react-router/dev/vite";
import { defineConfig } from "vite";
import { joinUrlPath } from "@plane/utils/string";

dotenv.config({ path: path.resolve(import.meta.dirname, ".env") });

// Expose only vars starting with VITE_
const viteEnv = Object.keys(process.env)
  .filter((k) => k.startsWith("VITE_"))
  .reduce<Record<string, string>>((a, k) => {
    a[k] = process.env[k] ?? "";
    return a;
  }, {});

const basePath = joinUrlPath(process.env.VITE_SPACE_BASE_PATH ?? "", "/") ?? "/";

export default defineConfig(() => ({
  base: basePath,
  define: {
    "process.env": JSON.stringify(viteEnv),
  },
  build: {
    assetsInlineLimit: 0,
  },
  plugins: [reactRouter()],
  resolve: {
    tsconfigPaths: true,
    dedupe: ["react", "react-dom"],
  },
  server: {
    host: process.env.PLANE_DEV_HOST || "127.0.0.1",
    port: Number(process.env.PLANE_SPACE_PORT ?? 3002),
    strictPort: true,
  },
}));
