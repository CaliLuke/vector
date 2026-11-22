import { expect, test } from "bun:test";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { lintFrontend } from "../observability-lint.mjs";

test("reports UI failure handling and detached rejections using the TypeScript AST", () => {
  const fixture = `
import { reportBrowserActionFailure as reportVectorAction } from "@/helpers/browser-diagnostics";
import { reportBrowserActionFailure } from "./unrelated-local-module";
type Action = () => Promise<void>;
export async function BrowserActions(save: Action) {
  try {
    await save();
  } catch (error) {
    setToast({ type: TOAST_TYPE.ERROR, message: "Could not save" }); // want unreported-ui-failure
  }

  try {
    await save();
  } catch (error) {
    if (isCancel(error)) return;
    console.error("Could not save");
    setFailure(error); // want local-only-ui-failure
  }

  try {
    await save();
  } catch (error) {
    if (isCancel(error)) console.warn("Expected cancellation");
    else setFailure(error); // want unreported-ui-failure
  }

  try {
    await save();
  } catch (error) {
    if (unrelated) queueMicrotask(() => console.error("Could not save", error));
    form.setErrorMap({ onSubmit: { form: "Save failed", fields: {} } }); // want unreported-ui-failure
  }

  try {
    await save();
  } catch (error) {
    setToast({ type: TOAST_TYPE.ERROR, title: "Try again" });
    throw error;
  }

  try {
    await save();
  } catch (error) {
    setFailure(error);
    throw new Error("Could not save", { cause: error });
  }

  try {
    await save();
  } catch (error) {
    setFailure(error);
    return Promise.reject(error);
  }

  try {
    await save();
  } catch (error) {
    setFailure(error); // want unreported-ui-failure
    throw new Error("Could not save");
  }

  try {
    await save();
  } catch {
    if (needsReset) form.setErrorMap({ onSubmit: undefined });
    setFailure(null);
    setToast({ type: TOAST_TYPE.SUCCESS });
  }

  try {
    await save();
  } catch (error) {
    void window.fetch("/api/diagnostics/project-access/", {
      method: "POST", credentials: "omit", keepalive: true, body: JSON.stringify({ stage: "identity", reason: "unexpected" })
    }).catch(() => {});
    setFailure(error);
  }

  try {
    await save();
  } catch (error) {
    reportVectorAction("theme_config_download");
    setFailure(error);
  }

  try {
    await save();
  } catch (error) {
    reportVectorAction("unexpected_action");
    setFailure(error); // want unreported-ui-failure
  }

  try {
    await save();
  } catch (error) {
    reportBrowserActionFailure("theme_config_download");
    setFailure(error); // want unreported-ui-failure
  }

  void save().catch(() => undefined); // want swallowed-detached-rejection
  void save().catch(() => { return undefined; }); // want swallowed-detached-rejection
  void save().catch(() => toast.error("Could not save")); // want unreported-ui-failure
  void client.invalidateQueries({ queryKey: keys }, { throwOnError: true }).catch(() => {}); // want swallowed-detached-rejection
  void client.invalidateQueries({ queryKey: keys }).catch(() => undefined);
  void client.cancelQueries({ queryKey: keys }).catch(() => undefined);
  void form.handleSubmit().catch(() => {});
  void save().catch((cause) => { if (!isCancel(cause)) throw cause; });
  return <span/>;
}
`;
  const directory = mkdtempSync(path.join(tmpdir(), "vector-obs-frontend-"));
  const file = path.join(directory, "fixture.tsx");
  try {
    writeFileSync(file, fixture);
    const matches = fixture.split("\n").flatMap((line, index) => {
      const rule = /\/\/ want ([a-z-]+)$/.exec(line)?.[1];
      return rule ? [{ line: index + 1, rule }] : [];
    });
    const findings = lintFrontend([file]);
    expect(findings.map(({ line, rule }) => ({ line, rule })).toSorted((a, b) => a.line - b.line)).toEqual(matches);
    expect(findings.every(({ column, message }) => column > 0 && typeof message === "string" && message.length)).toBe(
      true
    );
  } finally {
    rmSync(directory, { recursive: true, force: true });
  }
});
