export function seedInstanceInfo(client, config = {}) {
  client.setQueryData(["instance", "info"], { instance: { is_setup_done: true }, config });
}
