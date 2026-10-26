/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { MouseEvent } from "react";
import { FavoriteStar } from "@plane/ui";

type Props = {
  canFavorite: boolean;
  isReady: boolean;
  selected: boolean;
  onAdd: (event: MouseEvent<HTMLButtonElement>) => unknown;
  onRemove: (event: MouseEvent<HTMLButtonElement>) => unknown;
  disabled?: boolean;
  preventDefault?: boolean;
  stopPropagation?: boolean;
  buttonClassName?: string;
  iconClassName?: string;
};

export function AuthorizedFavoriteStar(props: Props) {
  const {
    canFavorite,
    isReady,
    selected,
    onAdd,
    onRemove,
    disabled = false,
    preventDefault = false,
    stopPropagation = false,
    buttonClassName,
    iconClassName,
  } = props;

  if (!canFavorite || !isReady || disabled) return null;

  return (
    <FavoriteStar
      buttonClassName={buttonClassName}
      iconClassName={iconClassName}
      onClick={(event) => {
        if (preventDefault) event.preventDefault();
        if (stopPropagation) event.stopPropagation();
        if (selected) void onRemove(event);
        else void onAdd(event);
      }}
      selected={selected}
    />
  );
}
