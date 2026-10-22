/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { EAuthModes } from "@plane/constants";

export function TermsAndConditions(_props: { authType?: EAuthModes }) {
  return (
    <p className="text-center text-13 text-tertiary">
      Vector is independent software derived from Plane. Your instance operator manages your account and data.{" "}
      <a
        className="underline"
        href="https://github.com/CaliLuke/vector/blob/main/NOTICE.md"
        target="_blank"
        rel="noopener noreferrer"
      >
        Attribution
      </a>
      {" · "}
      <a className="underline" href="https://github.com/CaliLuke/vector" target="_blank" rel="noopener noreferrer">
        Source code
      </a>
      {" · "}
      <a
        className="underline"
        href="https://github.com/CaliLuke/vector/blob/main/LICENSE.txt"
        target="_blank"
        rel="noopener noreferrer"
      >
        AGPL license · No warranty
      </a>
    </p>
  );
}
