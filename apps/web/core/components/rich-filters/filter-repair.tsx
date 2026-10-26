import { useState } from "react";
import { Button } from "@plane/propel/button";

export function FilterRepair(props: {
  message: string;
  canClear: boolean;
  disabled?: boolean;
  onClear: () => Promise<void>;
}) {
  const [pending, setPending] = useState(false);
  const [failed, setFailed] = useState(false);
  const clear = async () => {
    setPending(true);
    setFailed(false);
    try {
      await props.onClear();
    } catch {
      setFailed(true);
    } finally {
      setPending(false);
    }
  };
  return (
    <div role="alert" className="text-sm flex items-center gap-3 rounded-lg bg-layer-1 px-4 py-3">
      <span>{failed ? "Filters could not be cleared. Try again." : props.message}</span>
      {props.canClear && !props.disabled && (
        <Button
          variant="secondary"
          disabled={pending}
          onClick={() => {
            void clear();
          }}
        >
          Clear filters
        </Button>
      )}
    </div>
  );
}
