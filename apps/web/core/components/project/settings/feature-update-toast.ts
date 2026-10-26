import { setPromiseToast } from "@plane/propel/toast";

export function notifyProjectFeatureUpdate(update: Promise<unknown>) {
  setPromiseToast(update, {
    loading: "Updating project feature...",
    success: {
      title: "Success!",
      message: () => "Project feature updated successfully.",
    },
    error: {
      title: "Error!",
      message: () => "Something went wrong while updating project feature. Please try again.",
    },
  });
}
