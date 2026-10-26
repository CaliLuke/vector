import { useCallback, useRef } from "react";

/** Submit the latest draft after an active Form submission settles, without overlapping submissions. */
export function useAutosaveFormSubmit(form: {
  handleSubmit: () => Promise<void>;
  setErrorMap: (errors: { onSubmit: undefined }) => void;
}) {
  const active = useRef<Promise<void> | null>(null);
  const queued = useRef(false);
  const cancelQueued = useCallback(() => {
    queued.current = false;
  }, []);
  const submit = useCallback(
    function submit() {
      if (active.current) {
        queued.current = true;
        return;
      }
      form.setErrorMap({ onSubmit: undefined });
      // The form owns the submission error. Autosave is an event boundary, not another error surface.
      const pending = form.handleSubmit().catch(() => {});
      active.current = pending;
      void pending.then(() => {
        active.current = null;
        if (queued.current) {
          queued.current = false;
          submit();
        }
        return undefined;
      });
    },
    [form]
  );
  return { submit, cancelQueued };
}
