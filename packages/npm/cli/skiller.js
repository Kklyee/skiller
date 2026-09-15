#!/usr/bin/env node

"use strict";

const { spawnSync } = require("node:child_process");
const { existsSync } = require("node:fs");
const path = require("node:path");

const binaryName = process.platform === "win32" ? "skiller.exe" : "skiller";
const binaryPath = path.join(__dirname, "..", "bin", binaryName);

if (!existsSync(binaryPath)) {
  console.error("Skiller executable is missing. Reinstall skiller-cli to download it.");
  process.exitCode = 1;
} else {
  const result = spawnSync(binaryPath, process.argv.slice(2), {
    stdio: "inherit",
    windowsHide: true
  });

  if (result.error) {
    console.error(`Unable to start Skiller: ${result.error.message}`);
    process.exitCode = 1;
  } else if (result.signal) {
    process.exitCode = 1;
  } else {
    process.exitCode = typeof result.status === "number" ? result.status : 1;
  }
}
