/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { CircularProgressIndicator } from "@plane/ui";

export function AttachmentUploadProgress({ progress }: { progress: number }) {
  return (
    <div className="flex flex-shrink-0 items-center gap-2">
      <span className="flex-shrink-0">
        <CircularProgressIndicator size={20} strokeWidth={3} percentage={progress} />
      </span>
      <div className="flex-shrink-0 text-13 font-medium">{progress}% done</div>
    </div>
  );
}
