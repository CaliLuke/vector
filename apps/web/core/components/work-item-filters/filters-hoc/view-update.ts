import { setToast, TOAST_TYPE } from "@plane/propel/toast";

export function requireViewForUpdate<T>(view: T | null | undefined): NonNullable<T> {
  if (view) return view;
  setToast({
    type: TOAST_TYPE.ERROR,
    title: "We couldn't find the view",
    message: "The view you're trying to update doesn't exist.",
  });
  throw new Error("The view is no longer available.");
}

export async function persistViewUpdate(
  update: () => Promise<unknown>,
  isCurrent: () => boolean,
  acknowledge?: () => void
) {
  try {
    await update();
    if (!isCurrent()) throw new Error("The view changed while saving.");
    acknowledge?.();
    setToast({ type: TOAST_TYPE.SUCCESS, title: "Success!", message: "Your view has been updated successfully." });
  } catch (error) {
    if (isCurrent())
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Your view could not be updated. Please try again.",
      });
    throw error;
  }
}
