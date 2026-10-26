import { useMutation, useQueryClient } from "@tanstack/react-query";

/** Shared schema revisions invalidate workspace and project configuration together. */
export function useConfigurationMutation<Input, Result>(
  workspaceSlug: string,
  mutationFn: (input: Input) => Promise<Result>
) {
  const client = useQueryClient();
  const refresh = () => client.invalidateQueries({ queryKey: ["workspace", workspaceSlug] });
  return useMutation({
    mutationFn,
    // Invalidation only touches existing queries. Late writes after sign-out
    // cannot restore a cleared private catalog into another session.
    onSuccess: async () => {
      await refresh();
    },
    onError: () => {
      void refresh();
    },
    retry: false,
  });
}
