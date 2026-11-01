import assert from "node:assert/strict";
import { mkdtemp, writeFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();

await test("locale JSON readers reject non-object roots and include the source path in errors", async (t) => {
  const directory = await mkdtemp(path.join(tmpdir(), "vector-locale-input-"));
  t.after(() => rm(directory, { recursive: true, force: true }));
  const file = path.join(directory, "locale.json");
  const { readJsonFile } = await import("../../packages/i18n/scripts/lib/locale-io.ts");
  await writeFile(file, JSON.stringify({ nested: { message: "Hello" } }));
  assert.deepEqual(readJsonFile(file), { nested: { message: "Hello" } });
  await Promise.all(
    ["null", "[]", "42", '"text"', "invalid-json"].map(async (contents, index) => {
      const invalidFile = path.join(directory, `invalid-${index}.json`);
      await writeFile(invalidFile, contents);
      assert.throws(
        () => readJsonFile(invalidFile),
        (error) => error.message.includes(invalidFile)
      );
    })
  );
});
