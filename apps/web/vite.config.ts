import path from "node:path";
import * as dotenv from "dotenv";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

dotenv.config({ path: path.resolve(import.meta.dirname, ".env") });

// Expose only vars starting with VITE_
const viteEnv = Object.keys(process.env)
  .filter((k) => k.startsWith("VITE_"))
  .reduce<Record<string, string>>((a, k) => {
    a[k] = process.env[k] ?? "";
    return a;
  }, {});

export default defineConfig(() => ({
  define: {
    "process.env": JSON.stringify(viteEnv),
  },
  build: {
    assetsInlineLimit: 0,
    outDir: "build/client",
  },
  plugins: [tanstackRouter(), react()],
  resolve: {
    tsconfigPaths: true,
    dedupe: ["react", "react-dom", "@headlessui/react"],
  },
  server: {
    host: process.env.PLANE_DEV_HOST || "127.0.0.1",
    port: Number(process.env.PLANE_WEB_PORT ?? 3000),
    strictPort: true,
  },
}));
