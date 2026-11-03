import assert from "node:assert/strict";
import { once } from "node:events";
import { createRequire } from "./helpers/module-hooks.mjs";
import test from "node:test";
import {
  Controller,
  Get,
  Post,
  Middleware,
  WebSocket,
  registerController,
} from "../../packages/decorators/dist/index.mjs";

const liveRequire = createRequire(new URL("../../apps/live/package.json", import.meta.url));
const express = liveRequire("express");
const decoratorsRequire = createRequire(new URL("../../packages/decorators/package.json", import.meta.url));
const { defineMetadata } = decoratorsRequire("reflect-metadata/no-conflict");
const decorate = (decorator, Target, name) =>
  decorator(Target.prototype, name, Object.getOwnPropertyDescriptor(Target.prototype, name));

async function serve(t, router) {
  const app = express();
  app.use(router);
  app.use((error, _request, response, _next) => response.status(500).json({ message: error.message }));
  const server = app.listen(0, "127.0.0.1");
  t.after(() => new Promise((resolve, reject) => server.close((error) => (error ? reject(error) : resolve()))));
  await once(server, "listening");
  return `http://127.0.0.1:${server.address().port}`;
}

void test("REST decorators retain dependencies, middleware ordering, empty subroutes, and async error propagation", async (t) => {
  class Example {
    constructor(dependency) {
      this.dependency = dependency;
    }
    read(request, response) {
      response.json({ dependency: this.dependency, trace: request.trace });
    }
    async fail() {
      throw new Error("controlled failure");
    }
  }
  Controller("/example")(Example);
  decorate(Get(""), Example, "read");
  decorate(
    Middleware((request, _response, next) => {
      request.trace = ["first"];
      next();
    }),
    Example,
    "read"
  );
  decorate(
    Middleware((request, _response, next) => {
      request.trace.push("second");
      next();
    }),
    Example,
    "read"
  );
  decorate(Post("/failure"), Example, "fail");
  const router = express.Router();
  registerController(router, Example, ["injected"]);
  const base = await serve(t, router);
  assert.deepEqual(await (await fetch(`${base}/example`)).json(), {
    dependency: "injected",
    trace: ["first", "second"],
  });
  const failed = await fetch(`${base}/example/failure`, { method: "POST" });
  assert.equal(failed.status, 500);
  assert.deepEqual(await failed.json(), { message: "controlled failure" });
});

void test("decorating a derived method does not mutate its parent's middleware list", async (t) => {
  class Parent {
    read(request, response) {
      response.json(request.trace);
    }
  }
  class Child extends Parent {
    read(request, response) {
      super.read(request, response);
    }
  }
  Controller("/parent")(Parent);
  Controller("/child")(Child);
  decorate(Get(""), Parent, "read");
  decorate(
    Middleware((request, _response, next) => {
      request.trace = ["parent"];
      next();
    }),
    Parent,
    "read"
  );
  decorate(
    Middleware((request, _response, next) => {
      request.trace.push("child");
      next();
    }),
    Child,
    "read"
  );
  const router = express.Router();
  registerController(router, Parent);
  registerController(router, Child);
  const base = await serve(t, router);
  assert.deepEqual(await (await fetch(`${base}/parent`)).json(), ["parent"]);
  assert.deepEqual(await (await fetch(`${base}/child`)).json(), ["parent", "child"]);
});

void test("WebSocket controllers handle both synchronous and asynchronous failures without exposing their detail", async (t) => {
  const logs = t.mock.method(console, "error", () => {});
  const called = [];
  class SocketController {
    constructor(dependency) {
      this.dependency = dependency;
    }
    connect(_socket, request) {
      called.push([this.dependency, request]);
    }
    fail() {
      throw new Error("private connection detail");
    }
    async failAsync() {
      throw new Error("private asynchronous detail");
    }
  }
  Controller("/socket")(SocketController);
  for (const name of ["connect", "fail", "failAsync"]) decorate(WebSocket(`/${name}`), SocketController, name);
  const handlers = new Map();
  registerController({ ws: (route, handler) => handlers.set(route, handler) }, SocketController, ["dependency"]);
  const closed = [];
  const socket = { close: (...args) => closed.push(args) };
  const request = { header: "value" };
  handlers.get("/socket/connect")(socket, request);
  handlers.get("/socket/fail")(socket, request);
  handlers.get("/socket/failAsync")(socket, request);
  await Promise.resolve();
  assert.deepEqual(called, [["dependency", request]]);
  assert.deepEqual(closed, [
    [1011, "Internal server error"],
    [1011, "Internal server error"],
  ]);
  assert.equal(logs.mock.callCount(), 2);
});

void test("controller registration rejects malformed middleware and never dispatches arbitrary metadata methods", () => {
  class Example {
    read() {}
  }
  Controller("/example")(Example);
  decorate(Get("/read"), Example, "read");
  defineMetadata("middlewares", [42], Example.prototype, "read");
  assert.throws(() => registerController(express.Router(), Example), /Invalid controller middleware/);
  defineMetadata("method", "constructor", Example.prototype, "read");
  const router = express.Router();
  assert.doesNotThrow(() => registerController(router, Example));
  assert.equal(router.stack.length, 0);
  class MissingRoute {
    read() {}
  }
  assert.throws(() => registerController(router, MissingRoute), /Missing controller route/);
});
