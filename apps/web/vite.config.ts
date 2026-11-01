import path from "node:path";
import * as dotenv from "dotenv";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import { getFrontendViteConfig } from "../../scripts/vite-config.mts";

dotenv.config({ path: path.resolve(import.meta.dirname, ".env") });

export default defineConfig(() => ({
  ...getFrontendViteConfig({ env: process.env, portVariable: "PLANE_WEB_PORT", defaultPort: 3000 }),
  build: { assetsInlineLimit: 0, outDir: "build/client" },
  plugins: [tanstackRouter(), react()],
  resolve: {
    tsconfigPaths: true,
    dedupe: ["react", "react-dom", "@headlessui/react"],
  },
}));
