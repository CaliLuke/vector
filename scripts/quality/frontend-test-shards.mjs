import { statSync, writeFileSync } from "node:fs";
import path from "node:path";

// Source size is a stable cost estimate. Spread larger tests first rather than
// letting alternating filenames put most expensive files in the same shard.
export function partitionTestFiles(files) {
  const shards = [[], []];
  const sizes = [0, 0];
  const weighted = files.map((file) => ({ file, size: statSync(file).size }));
  weighted.sort((left, right) => right.size - left.size || left.file.localeCompare(right.file));
  for (const { file, size } of weighted) {
    const shard = sizes[0] <= sizes[1] ? 0 : 1;
    shards[shard].push(file);
    sizes[shard] += size;
  }
  return shards;
}

if (import.meta.main) {
  const [directory, ...files] = process.argv.slice(2);
  for (const [index, shard] of partitionTestFiles(files).entries()) {
    writeFileSync(path.join(directory, `${index + 1}`), shard.map((file) => `${file}\0`).join(""));
  }
}
