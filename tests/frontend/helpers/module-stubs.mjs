export function createModuleStub(source) {
  return {
    shortCircuit: true,
    url: `data:text/javascript,${encodeURIComponent(source)}`,
  };
}

export function resolvePackageStub(specifier, packageStubs, context, nextResolve) {
  if (Object.hasOwn(packageStubs, specifier)) return createModuleStub(packageStubs[specifier]);
  return nextResolve(specifier, context);
}
