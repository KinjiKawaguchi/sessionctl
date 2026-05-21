#!/usr/bin/env node
const { execFileSync } = require("child_process");
const path = require("path");
const os = require("os");

const ext = os.platform() === "win32" ? ".exe" : "";
const bin = path.join(__dirname, "bin", "sessionctl" + ext);

try {
  execFileSync(bin, process.argv.slice(2), { stdio: "inherit" });
} catch (e) {
  if (e.status != null) process.exit(e.status);
  console.error("sessionctl: binary not found. Try reinstalling: npm install sessionctl");
  process.exit(1);
}
