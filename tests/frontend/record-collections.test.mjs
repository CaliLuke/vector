import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createQueryDomFixture } from "./helpers/query-dom.mjs";
import { React, act } from "./helpers/react.mjs";

registerWebModuleHooks();
const { useRecordCollection } = await import("../../packages/hooks/src/use-record-collection.ts");
const { getRecordCollection } = await import("../../packages/hooks/src/record-collection.ts");
const { QueryClient } = await import("@tanstack/react-query");

function observer(options, receive) {
  return function Observer() {
    const records = useRecordCollection(options);
    receive(records);
    return React.createElement("output", null, records.data?.map((row) => row.name).join(",") ?? "unavailable");
  };
}

function probe(options) {
  let current;
  return {
    Component: observer(options, (value) => {
      current = value;
    }),
    get current() {
      return current;
    },
  };
}

await test("live records share one transport and follow replacements, reordering, and deletion", async (t) => {
  const fixture = await createQueryDomFixture(t);
  let calls = 0;
  const options = {
    queryKey: ["records", "alice", "workspace"],
    staleTime: Infinity,
    queryFn: async () => {
      calls++;
      return [
        { id: "a", name: "A" },
        { id: "b", name: "B" },
      ];
    },
  };
  const first = probe(options),
    second = probe(options);
  await fixture.render(
    React.createElement(
      React.StrictMode,
      null,
      React.createElement(first.Component),
      React.createElement(second.Component)
    )
  );
  assert.equal(calls, 1);
  assert.equal(fixture.client.getQueryCache().findAll({ queryKey: options.queryKey }).length, 1);
  assert.deepEqual(first.current.data, [
    { id: "a", name: "A" },
    { id: "b", name: "B" },
  ]);

  await act(async () =>
    fixture.client.setQueryData(options.queryKey, [
      { id: "b", name: "New B" },
      { id: "a", name: "A" },
    ])
  );
  assert.deepEqual(first.current.data, [
    { id: "b", name: "New B" },
    { id: "a", name: "A" },
  ]);
  assert.deepEqual([...first.current.byId.keys()], ["b", "a"], "lookup iteration follows live row order");
  assert.equal(second.current.get("b").name, "New B");
  await act(async () => fixture.client.setQueryData(options.queryKey, []));
  assert.deepEqual(first.current.data, []);
  assert.equal(second.current.get("a"), undefined);
});

await test("failed refreshes retain the last snapshot but invalidate lookups until retry", async (t) => {
  const fixture = await createQueryDomFixture(t);
  let fail = false;
  const options = {
    queryKey: ["records", "errors"],
    staleTime: Infinity,
    queryFn: async () => {
      if (fail) throw new Error("denied");
      return [{ id: "a", name: "Allowed" }];
    },
  };
  let result;
  await fixture.render(
    React.createElement(
      observer(options, (value) => {
        result = value;
      })
    )
  );
  assert.equal(result.get("a").name, "Allowed");
  fail = true;
  await act(async () => result.refetch());
  assert.equal(result.isError, true);
  assert.equal(result.data[0].name, "Allowed");
  assert.equal(result.get("a"), undefined);
  fail = false;
  await act(async () => result.refetch({ throwOnError: true }));
  assert.equal(result.isSuccess, true);
  assert.equal(result.get("a").name, "Allowed");
});

await test("collections cannot cross client or exact query ownership", async (t) => {
  const { client } = await createQueryDomFixture(t);
  const otherClient = new QueryClient();
  t.after(() => {
    otherClient.clear();
  });
  const options = { queryKey: ["records", "same-key"], queryFn: async () => [], staleTime: Infinity };
  client.setQueryData(options.queryKey, [{ id: "secret", name: "Private" }]);
  const first = getRecordCollection(client, options, (row) => row.id).collection;
  const other = getRecordCollection(otherClient, options, (row) => row.id).collection;
  await first.preload();
  assert.equal(first.get("secret").value.name, "Private");
  assert.notEqual(first, other);
  client.removeQueries({ queryKey: options.queryKey });
  const replacement = getRecordCollection(client, options, (row) => row.id).collection;
  assert.notEqual(first, replacement);
  await new Promise((resolve) => setTimeout(resolve, 0));
  assert.equal(first.size, 0);
  assert.equal(replacement.size, 0);
});

await test("disabled collections retain their snapshot without loading or offering current lookups", async (t) => {
  const fixture = await createQueryDomFixture(t);
  const options = {
    queryKey: ["records", "disabled"],
    enabled: false,
    queryFn: async () => assert.fail("disabled read"),
  };
  fixture.client.setQueryData(options.queryKey, [{ id: "a", name: "Private" }]);
  let result;
  await fixture.render(
    React.createElement(
      observer(options, (value) => {
        result = value;
      })
    )
  );
  assert.deepEqual(result.data, [{ id: "a", name: "Private" }]);
  assert.equal(result.get("a"), undefined);
  assert.equal(fixture.client.getQueryCache().find({ queryKey: options.queryKey }).isActive(), false);
  assert.equal(result.isPending, true);
  assert.equal(result.isLoading, false);
});

await test("leaving a collection releases its observer but preserves an accepted write for remount", async (t) => {
  const fixture = await createQueryDomFixture(t, { defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  const options = {
    queryKey: ["records", "navigation"],
    queryFn: async () => [{ id: "a", name: "Before" }],
    staleTime: Infinity,
  };
  let result;
  const Probe = observer(options, (value) => {
    result = value;
  });
  await fixture.render(React.createElement(Probe));
  const owner = fixture.client.getQueryCache().find({ queryKey: options.queryKey });
  await fixture.render(null);
  await new Promise((resolve) => setTimeout(resolve, 25));
  assert.equal(owner.getObserversCount(), 0);
  assert.equal(fixture.client.getQueryCache().find({ queryKey: options.queryKey }), owner);
  fixture.client.setQueryData(options.queryKey, [{ id: "a", name: "Saved after navigation" }]);
  await fixture.render(React.createElement(Probe));
  assert.equal(result.get("a").name, "Saved after navigation");
});

await test("a failed initial load retries its live view without losing the Query owner", async (t) => {
  const fixture = await createQueryDomFixture(t);
  let failed = true;
  const options = {
    queryKey: ["records", "initial-failure"],
    retry: false,
    queryFn: async () => {
      if (failed) throw new Error("offline");
      return [{ id: "a", name: "Recovered" }];
    },
  };
  let result;
  await fixture.render(
    React.createElement(
      observer(options, (value) => {
        result = value;
      })
    )
  );
  assert.equal(result.isError, true);
  const owner = fixture.client.getQueryCache().find({ queryKey: options.queryKey });
  failed = false;
  await act(async () => result.refetch({ throwOnError: true }));
  assert.equal(result.get("a").name, "Recovered");
  assert.equal(fixture.client.getQueryCache().find({ queryKey: options.queryKey }), owner);
});

await test("a retired refetch cannot recreate a removed private query", async (t) => {
  const fixture = await createQueryDomFixture(t);
  const options = {
    queryKey: ["records", "retired"],
    staleTime: Infinity,
    queryFn: async () => [{ id: "private", name: "Private" }],
  };
  let result;
  await fixture.render(
    React.createElement(
      observer(options, (value) => {
        result = value;
      })
    )
  );
  const retired = result;
  await fixture.render(null);
  fixture.client.removeQueries({ queryKey: options.queryKey, exact: true });
  await retired.refetch();
  assert.equal(retired.get("private"), undefined);
  assert.equal(fixture.client.getQueryCache().find({ queryKey: options.queryKey, exact: true }), undefined);
});

await test("a new consumer refreshes stale records while an existing consumer stays mounted", async (t) => {
  const fixture = await createQueryDomFixture(t);
  let name = "Before";
  const options = {
    queryKey: ["records", "mount-freshness"],
    staleTime: 0,
    refetchOnMount: true,
    queryFn: async () => [{ id: "a", name }],
  };
  const first = probe(options),
    second = probe(options);
  await fixture.render(
    React.createElement(React.Fragment, null, React.createElement(first.Component, { key: "first" }))
  );
  assert.equal(first.current.get("a").name, "Before");
  name = "After";
  await fixture.render(
    React.createElement(
      React.Fragment,
      null,
      React.createElement(first.Component, { key: "first" }),
      React.createElement(second.Component, { key: "second" })
    )
  );
  assert.equal(first.current.get("a").name, "After");
  assert.equal(second.current.get("a").name, "After");
});

await test("mount policies respect fresh, always and static data", async (t) => {
  const fixture = await createQueryDomFixture(t);
  const options = {
    queryKey: ["records", "mount-policies"],
    staleTime: Infinity,
    queryFn: async () => [{ id: "a", name: "Refetched" }],
  };
  fixture.client.setQueryData(options.queryKey, [{ id: "a", name: "Cached" }]);
  let result;
  const Probe = ({ policy, enabled = true }) => {
    const value = useRecordCollection({ ...options, ...policy, enabled });
    React.useLayoutEffect(() => {
      result = value;
    }, [value]);
    return null;
  };
  await fixture.render(React.createElement(Probe));
  assert.equal(result.get("a").name, "Cached");
  await fixture.render(
    React.createElement(Probe, { key: "no-refresh", policy: { staleTime: 0, refetchOnMount: false } })
  );
  assert.equal(result.get("a").name, "Cached");
  await fixture.render(React.createElement(Probe, { key: "always", policy: { refetchOnMount: "always" } }));
  assert.equal(result.get("a").name, "Refetched");
  await act(async () => fixture.client.setQueryData(options.queryKey, [{ id: "a", name: "Static" }]));
  await fixture.render(
    React.createElement(Probe, { key: "static", policy: { staleTime: "static", refetchOnMount: "always" } })
  );
  assert.equal(result.get("a").name, "Static");
});

await test("retryOnMount false does not retry a failed collection until explicitly requested", async (t) => {
  const fixture = await createQueryDomFixture(t);
  const options = {
    queryKey: ["records", "retry-on-mount"],
    retry: false,
    retryOnMount: false,
    queryFn: async () => [{ id: "a", name: "Recovered" }],
  };
  await assert.rejects(
    fixture.client.fetchQuery({
      ...options,
      queryFn: async () => {
        throw new Error("offline");
      },
    })
  );
  let result;
  await fixture.render(
    React.createElement(
      observer(options, (value) => {
        result = value;
      })
    )
  );
  assert.equal(result.isError, true);
  assert.equal(result.isFetching, false);
  await act(async () => result.refetch());
  assert.equal(result.get("a").name, "Recovered");
});

await test("enabling an existing consumer loads its collection", async (t) => {
  const fixture = await createQueryDomFixture(t);
  const options = { queryKey: ["records", "enable"], queryFn: async () => [{ id: "a", name: "Enabled" }] };
  let result;
  function Probe({ enabled }) {
    const value = useRecordCollection({ ...options, enabled });
    React.useLayoutEffect(() => {
      result = value;
    }, [value]);
    return null;
  }
  await fixture.render(React.createElement(Probe, { enabled: false }));
  assert.equal(result.data, undefined);
  await fixture.render(React.createElement(Probe, { enabled: true }));
  assert.equal(result.get("a").name, "Enabled");
});

await test("a captured refetch cannot target the replacement consumer's query", async (t) => {
  const fixture = await createQueryDomFixture(t);
  let secondName = "Second";
  let result;
  const firstOptions = {
    queryKey: ["records", "first-scope"],
    staleTime: Infinity,
    queryFn: async () => [{ id: "a", name: "First" }],
  };
  const secondOptions = {
    queryKey: ["records", "second-scope"],
    staleTime: Infinity,
    queryFn: async () => [{ id: "a", name: secondName }],
  };
  function Probe({ options }) {
    const value = useRecordCollection(options);
    React.useLayoutEffect(() => {
      result = value;
    }, [value]);
    return null;
  }
  await fixture.render(React.createElement(Probe, { options: firstOptions }));
  const first = result;
  await fixture.render(React.createElement(Probe, { options: secondOptions }));
  secondName = "Should not load";
  await act(async () => first.refetch());
  assert.equal(result.get("a").name, "Second");
});

await test("a departed static consumer cannot prevent the remaining consumer from refreshing", async (t) => {
  const fixture = await createQueryDomFixture(t);
  let name = "Before";
  const options = { queryKey: ["records", "departed-static"], queryFn: async () => [{ id: "a", name }] };
  const fixed = probe({ ...options, staleTime: "static" });
  const normal = probe({ ...options, staleTime: Infinity });
  const render = (includeFixed) =>
    fixture.render(
      React.createElement(
        React.Fragment,
        null,
        includeFixed && React.createElement(fixed.Component, { key: "fixed" }),
        React.createElement(normal.Component, { key: "normal" })
      )
    );
  await fixture.render(
    React.createElement(React.Fragment, null, React.createElement(fixed.Component, { key: "fixed" }))
  );
  await render(true);
  await render(false);
  name = "After";
  await act(async () => fixture.client.invalidateQueries({ queryKey: options.queryKey }));
  assert.equal(normal.current.get("a").name, "After");
});
