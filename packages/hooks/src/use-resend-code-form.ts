/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useEffect, useState } from "react";
import { useForm } from "@tanstack/react-form";

type Props = { send: () => Promise<void>; onSent: () => void; onError: () => void };

export function useResendCodeForm({ send, onSent, onError }: Props) {
  const [cooldown, setCooldown] = useState(0);
  useEffect(() => {
    if (cooldown <= 0) return undefined;
    const timeout = setTimeout(() => setCooldown((remaining) => remaining - 1), 1000);
    return () => clearTimeout(timeout);
  }, [cooldown]);
  const form = useForm({
    defaultValues: {},
    onSubmit: async () => {
      if (cooldown > 0) return;
      try {
        await send();
        onSent();
        setCooldown(5);
      } catch (error) {
        onError();
        throw error;
      }
    },
  });
  return { form, cooldown };
}
