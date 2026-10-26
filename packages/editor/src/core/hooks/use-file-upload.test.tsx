/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { useStore } from "@tanstack/react-form";
import { expect, it, vi } from "vitest";
import { useUploader } from "./use-file-upload";

function deferredUpload() {
  let resolve!: (assetId: string) => void;
  const promise = new Promise<string>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}

type TUploadAttempt = ReturnType<typeof useUploader>["attempts"][number];

function getErrorMessage(error: unknown) {
  if (error && typeof error === "object" && "message" in error && typeof error.message === "string")
    return error.message;
  return "";
}

function AttemptState({
  attempt,
  retryUpload,
}: {
  attempt: TUploadAttempt;
  retryUpload: ReturnType<typeof useUploader>["retryUpload"];
}) {
  const state = useStore(attempt.form.store, (formState) => ({
    error: formState.errorMap.onSubmit,
    isSubmitSuccessful: formState.isSubmitSuccessful,
    isSubmitting: formState.isSubmitting,
  }));
  const status = state.isSubmitSuccessful
    ? "uploaded"
    : state.isSubmitting
      ? "uploading"
      : state.error && typeof state.error === "object" && "kind" in state.error && state.error.kind === "validation"
        ? "invalid"
        : state.error && typeof state.error === "object" && "kind" in state.error && state.error.kind === "transport"
          ? "failed"
          : "ready";
  const error = getErrorMessage(state.error);
  return (
    <span data-status={status}>
      {attempt.form.state.values.file.name}:{error}
      {status === "failed" && <button onClick={() => void retryUpload(attempt.id)}>Retry</button>}
    </span>
  );
}

it("finishing one upload does not clear a control's busy state while another upload is pending", async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  const first = deferredUpload();
  const second = deferredUpload();
  const command = vi.fn().mockReturnValueOnce(first.promise).mockReturnValueOnce(second.promise);
  const handleProgressStatus = vi.fn();
  const onUpload = vi.fn();
  const file = new File(["image"], "image.png", { type: "image/png" });
  function Harness() {
    const uploader = useUploader({
      acceptedMimeTypes: ["image/png"],
      editorCommand: command,
      maxFileSize: 1000,
      handleProgressStatus,
      onUpload,
    });
    const { isUploading, uploadFile } = uploader;
    return <button onClick={() => void uploadFile(file)}>{isUploading ? "Uploading" : "Ready"}</button>;
  }
  const host = document.createElement("div");
  document.body.append(host);
  const root = createRoot(host);
  try {
    await act(async () => root.render(<Harness />));
    const button = host.querySelector("button");
    if (!button) throw new Error("Upload button did not mount");
    await act(async () => {
      button.click();
      button.click();
    });
    expect(button.textContent).toBe("Uploading");
    expect(handleProgressStatus).toHaveBeenCalledWith(true);
    await act(async () => first.resolve("first"));
    expect(button.textContent).toBe("Uploading");
    expect(handleProgressStatus).not.toHaveBeenCalledWith(false);
    await act(async () => second.resolve("second"));
    expect(button.textContent).toBe("Ready");
    expect(handleProgressStatus).toHaveBeenLastCalledWith(false);
    expect(onUpload).toHaveBeenCalledWith("first", file);
    expect(onUpload).toHaveBeenCalledWith("second", file);
  } finally {
    await act(async () => root.unmount());
    host.remove();
  }
});

it("keeps validation and failed-file retry state in the mounted native uploader form", async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  const command = vi.fn().mockRejectedValueOnce(new Error("offline")).mockResolvedValueOnce("saved-image");
  const onUpload = vi.fn();
  const invalidFile = new File(["text"], "notes.txt", { type: "text/plain" });
  const validFile = new File(["image"], "image.png", { type: "image/png" });
  function Harness() {
    const uploader = useUploader({
      acceptedMimeTypes: ["image/png"],
      editorCommand: command,
      maxFileSize: 1000,
      onUpload,
    });
    return (
      <>
        <button onClick={() => void uploader.uploadFile(invalidFile)}>Select invalid</button>
        <button onClick={() => void uploader.uploadFile(validFile)}>Select valid</button>
        <output>
          {uploader.attempts.map((attempt) => (
            <AttemptState key={attempt.id} attempt={attempt} retryUpload={uploader.retryUpload} />
          ))}
        </output>
      </>
    );
  }
  const host = document.createElement("div");
  document.body.append(host);
  const root = createRoot(host);
  try {
    await act(async () => root.render(<Harness />));
    const selectInvalid = [...host.querySelectorAll("button")].find(
      (button) => button.textContent === "Select invalid"
    );
    if (!selectInvalid) throw new Error("Invalid-file control did not render");
    await act(async () => selectInvalid.click());
    expect(host.querySelector('[data-status="invalid"]')?.textContent).toContain("Invalid file type.");
    expect(command).not.toHaveBeenCalled();

    const selectValid = [...host.querySelectorAll("button")].find((button) => button.textContent === "Select valid");
    if (!selectValid) throw new Error("Valid-file control did not render");
    await act(async () => selectValid.click());
    expect(host.querySelector('[data-status="failed"]')?.textContent).toContain("image.png:offline");
    expect(onUpload).not.toHaveBeenCalled();

    const retry = [...host.querySelectorAll("button")].find((button) => button.textContent === "Retry");
    if (!retry) throw new Error("Failed upload did not expose a retry control");
    await act(async () => retry.click());
    expect(host.querySelector('[data-status="failed"]')).toBeNull();
    expect(host.querySelector('[data-status="uploaded"]')?.textContent).toBe("image.png:");
    expect(command).toHaveBeenCalledTimes(2);
    expect(onUpload).toHaveBeenCalledExactlyOnceWith("saved-image", validFile);
  } finally {
    await act(async () => root.unmount());
    host.remove();
  }
});
