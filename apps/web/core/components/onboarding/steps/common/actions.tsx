/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMutation } from "@tanstack/react-query";
import { Button } from "@plane/propel/button";

type Props = { isSubmitting: boolean; canSubmit: boolean; onSkip: () => Promise<void>; skipLabel?: string };

export function OnboardingActions({ isSubmitting, canSubmit, onSkip, skipLabel = "Skip" }: Props) {
  const skip = useMutation({ mutationFn: onSkip });
  const busy = isSubmitting || skip.isPending;
  return (
    <div className="space-y-3">
      <Button
        variant="primary"
        type="submit"
        className="w-full"
        size="xl"
        disabled={!canSubmit || busy}
        loading={isSubmitting}
      >
        Continue
      </Button>
      <Button
        type="button"
        variant="ghost"
        className="w-full text-tertiary"
        size="xl"
        disabled={busy}
        loading={skip.isPending}
        onClick={() => skip.mutate()}
      >
        {skipLabel}
      </Button>
      {skip.isError && (
        <p role="alert" className="text-13 text-danger-primary">
          Unable to continue. Please try again.
        </p>
      )}
    </div>
  );
}
