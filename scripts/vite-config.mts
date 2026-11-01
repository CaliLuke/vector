/** Shared frontend environment and development-server policy. */
export function getFrontendViteConfig({
  env,
  portVariable,
  defaultPort,
}: {
  env: Record<string, string | undefined>;
  portVariable: string;
  defaultPort: number;
}) {
  const publicEnvironment = Object.fromEntries(
    Object.entries(env)
      .filter(([name]) => name.startsWith("VITE_"))
      .map(([name, value]) => [name, value ?? ""])
  );
  return {
    define: { "process.env": JSON.stringify(publicEnvironment) },
    build: { assetsInlineLimit: 0 },
    resolve: { tsconfigPaths: true, dedupe: ["react", "react-dom"] },
    server: {
      host: env.PLANE_DEV_HOST || "127.0.0.1",
      port: Number(env[portVariable] ?? defaultPort),
      strictPort: true,
    },
  };
}
