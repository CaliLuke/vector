/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export function AttachmentUploadError({
  error,
  canRetry,
  disabled,
  retry,
}: {
  error: string | undefined;
  canRetry: boolean;
  disabled: boolean;
  retry: () => Promise<void>;
}) {
  if (!error) return null;
  return (
    <div role="alert" className="flex items-center gap-2 text-11 text-danger-primary">
      <span>{error}</span>
      {canRetry && (
        <button type="button" disabled={disabled} onClick={() => void retry()}>
          Retry upload
        </button>
      )}
    </div>
  );
}
