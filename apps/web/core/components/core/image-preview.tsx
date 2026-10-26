/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useEffect, useRef } from "react";
import type { ComponentProps } from "react";

type Props = ComponentProps<"img"> & { file: File | null };

export function ImagePreview({ file, src, alt, ...props }: Props) {
  const image = useRef<HTMLImageElement>(null);
  useEffect(() => {
    if (!file || !image.current) return undefined;
    const url = URL.createObjectURL(file);
    image.current.src = url;
    return () => URL.revokeObjectURL(url);
  }, [file]);
  return <img {...props} ref={image} src={file ? undefined : src} alt={alt} />;
}
