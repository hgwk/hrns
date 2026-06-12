#!/usr/bin/env node
"use strict";

const fs = require("fs");
const https = require("https");
const crypto = require("crypto");
const os = require("os");
const path = require("path");
const { spawnSync } = require("child_process");

const pkg = require("../package.json");
const platformMap = { darwin: "darwin", linux: "linux" };
const archMap = { x64: "amd64", arm64: "arm64" };

const goos = platformMap[os.platform()];
const goarch = archMap[os.arch()];
if (!goos || !goarch) {
  console.error(`hrns: unsupported platform ${os.platform()}/${os.arch()}`);
  process.exit(1);
}

const version = pkg.version;
const asset = `hrns_${version}_${goos}_${goarch}.tar.gz`;
const url = `https://github.com/hgwk/hrns/releases/download/v${version}/${asset}`;
const sumsUrl = `https://github.com/hgwk/hrns/releases/download/v${version}/SHA256SUMS`;
const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "hrns-install-"));
const archive = path.join(tmp, asset);
const outDir = path.join(__dirname, "native");

function download(targetUrl, dest, redirects = 0) {
  if (redirects > 5) throw new Error("too many redirects");
  return new Promise((resolve, reject) => {
    https.get(targetUrl, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        res.resume();
        download(res.headers.location, dest, redirects + 1).then(resolve, reject);
        return;
      }
      if (res.statusCode !== 200) {
        res.resume();
        reject(new Error(`download failed: HTTP ${res.statusCode}`));
        return;
      }
      const file = fs.createWriteStream(dest);
      res.pipe(file);
      file.on("finish", () => file.close(resolve));
      file.on("error", reject);
    }).on("error", reject);
  });
}

function getText(targetUrl, redirects = 0) {
  if (redirects > 5) throw new Error("too many redirects");
  return new Promise((resolve, reject) => {
    https.get(targetUrl, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        res.resume();
        getText(res.headers.location, redirects + 1).then(resolve, reject);
        return;
      }
      if (res.statusCode !== 200) {
        res.resume();
        reject(new Error(`download failed: HTTP ${res.statusCode}`));
        return;
      }
      let body = "";
      res.setEncoding("utf8");
      res.on("data", (chunk) => { body += chunk; });
      res.on("end", () => resolve(body));
      res.on("error", reject);
    }).on("error", reject);
  });
}

async function verifyChecksum(file) {
  const sums = await getText(sumsUrl);
  const row = sums
    .split(/\r?\n/)
    .map((line) => line.trim().split(/\s+/))
    .find((parts) => parts[1] === asset);
  if (!row) throw new Error(`checksum missing for ${asset}`);
  const actual = crypto.createHash("sha256").update(fs.readFileSync(file)).digest("hex");
  if (actual !== row[0]) throw new Error(`checksum mismatch for ${asset}`);
}

(async () => {
  fs.mkdirSync(outDir, { recursive: true });
  await download(url, archive);
  await verifyChecksum(archive);
  const tar = spawnSync("tar", ["-xzf", archive, "-C", tmp], { stdio: "inherit" });
  if (tar.status !== 0) process.exit(tar.status || 1);
  const unpacked = path.join(tmp, `hrns_${version}_${goos}_${goarch}`, "hrns");
  const target = path.join(outDir, "hrns");
  fs.copyFileSync(unpacked, target);
  fs.chmodSync(target, 0o755);
})().catch((err) => {
  console.error(`hrns install failed: ${err.message}`);
  console.error("Install Go and run `go install github.com/hgwk/hrns/cmd/hrns@latest` as a fallback.");
  process.exit(1);
});
