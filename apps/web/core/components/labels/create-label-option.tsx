/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { KeyboardEventHandler } from "react";

type Props = {
  query: string;
  isSubmitting: boolean;
  emptyLabel: string;
  onCreate: () => void;
  onKeyDown?: KeyboardEventHandler<HTMLButtonElement>;
};

export function CreateLabelOption({ query, isSubmitting, emptyLabel, onCreate, onKeyDown }: Props) {
  return (
    <button
      type="button"
      disabled={!query.trim() || isSubmitting}
      onClick={onCreate}
      onKeyDown={onKeyDown}
      className={`text-left text-secondary ${query.length ? "cursor-pointer" : "cursor-default"}`}
    >
      {/* TODO: translate here */}
      {query.length ? (
        <>
          + Add <span className="text-primary">&quot;{query}&quot;</span> to labels
        </>
      ) : (
        emptyLabel
      )}
    </button>
  );
}
