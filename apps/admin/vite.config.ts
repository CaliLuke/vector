import path from "node:path";
import * as dotenv from "dotenv";
import { reactRouter } from "@react-router/dev/vite";
import { defineConfig } from "vite";
import { getFrontendViteConfig } from "../../scripts/vite-config.mts";
import { joinUrlPath } from "@plane/utils/string";

dotenv.config({ path: path.resolve(import.meta.dirname, ".env") });

const basePath = joinUrlPath(process.env.VITE_ADMIN_BASE_PATH ?? "", "/") ?? "/";

export default defineConfig(() => ({
  base: basePath,
  ...getFrontendViteConfig({ env: process.env, portVariable: "PLANE_ADMIN_PORT", defaultPort: 3001 }),
  plugins: [reactRouter()],
}));
