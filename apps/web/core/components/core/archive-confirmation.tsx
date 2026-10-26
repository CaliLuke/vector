/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { Button } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { EModalPosition, EModalWidth, ModalCore } from "@plane/ui";
import { useCommittedScope } from "@/hooks/use-committed-scope";

type ArchiveResult = { captured: { isCurrent: () => boolean } };
type Props = {
  name: string | undefined;
  noun: "cycle" | "module";
  scopeKey: string;
  isPending: boolean;
  archive: () => Promise<ArchiveResult>;
  isCurrent: () => boolean;
  onClose: () => void;
  onArchived: () => void;
};
/** Shared confirmation behavior: failed writes stay open, stale completions have no dialog effects. */
export function ArchiveConfirmation({
  name,
  noun,
  scopeKey,
  isPending,
  archive,
  isCurrent,
  onClose,
  onArchived,
}: Props) {
  const mounted = useCommittedScope(scopeKey);
  const submit = async () => {
    try {
      const result = await archive();
      if (!mounted() || !result.captured.isCurrent()) return;
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Archive success",
        message: "Your archives can be found in project archives.",
      });
      onArchived();
    } catch {
      if (mounted() && isCurrent())
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message: `The ${noun} could not be archived. Please try again.`,
        });
    }
  };
  return (
    <ModalCore isOpen handleClose={onClose} position={EModalPosition.CENTER} width={EModalWidth.LG}>
      <div className="px-5 py-4">
        <h3 className="text-18 font-medium 2xl:text-20">
          Archive {noun} {name}
        </h3>
        <p className="mt-3 text-13 text-secondary">
          Are you sure you want to archive the {noun}? All your archives can be restored later.
        </p>
        <div className="mt-3 flex justify-end gap-2">
          <Button variant="secondary" size="lg" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="primary" size="lg" onClick={() => void submit()} loading={isPending}>
            {isPending ? "Archiving" : "Archive"}
          </Button>
        </div>
      </div>
    </ModalCore>
  );
}
