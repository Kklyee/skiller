"use strict";

const fs = require("node:fs");
const http = require("node:http");
const https = require("node:https");
const path = require("node:path");
const { URL } = require("node:url");
const zlib = require("node:zlib");

const packageRoot = path.resolve(__dirname, "..");
const packageJSON = JSON.parse(fs.readFileSync(path.join(packageRoot, "package.json"), "utf8"));
const platformMap = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows"
};
const architectureMap = {
  arm64: "arm64",
  x64: "amd64"
};

function readTarField(buffer, offset, length) {
  return buffer.subarray(offset, offset + length).toString("utf8").replace(/\0.*$/, "").trim();
}

function download(url, redirects = 0) {
  if (redirects > 5) {
    return Promise.reject(new Error("too many redirects while downloading the release archive"));
  }

  return new Promise((resolve, reject) => {
    const parsedURL = new URL(url);
    const transport = parsedURL.protocol === "http:" ? http : https;
    const request = transport.get(parsedURL, {
      headers: {
        "User-Agent": "skiller-cli-npm"
      }
    }, (response) => {
      const statusCode = response.statusCode || 0;
      if (statusCode >= 300 && statusCode < 400 && response.headers.location) {
        response.resume();
        resolve(download(new URL(response.headers.location, parsedURL).toString(), redirects + 1));
        return;
      }

      if (statusCode !== 200) {
        response.resume();
        reject(new Error(`release archive download returned HTTP ${statusCode}`));
        return;
      }

      const chunks = [];
      response.on("data", (chunk) => chunks.push(chunk));
      response.on("end", () => resolve(Buffer.concat(chunks)));
      response.on("error", reject);
    });

    request.on("error", reject);
  });
}

function extractBinary(archive, binaryName) {
  const tarball = zlib.gunzipSync(archive);
  let offset = 0;

  while (offset + 512 <= tarball.length) {
    const header = tarball.subarray(offset, offset + 512);
    if (header.every((byte) => byte === 0)) {
      break;
    }

    const name = readTarField(header, 0, 100);
    const prefix = readTarField(header, 345, 155);
    const entryName = prefix ? `${prefix}/${name}` : name;
    const sizeText = readTarField(header, 124, 12);
    const size = sizeText ? parseInt(sizeText, 8) : 0;
    const dataStart = offset + 512;
    const dataEnd = dataStart + size;

    if (!Number.isSafeInteger(size) || dataEnd > tarball.length) {
      throw new Error("release archive contains an invalid tar entry");
    }

    if ((header[156] === 0 || header[156] === 48) && entryName === binaryName) {
      return tarball.subarray(dataStart, dataEnd);
    }

    offset = dataStart + Math.ceil(size / 512) * 512;
  }

  throw new Error(`release archive does not contain ${binaryName}`);
}

function targetForCurrentMachine() {
  const goos = platformMap[process.platform];
  const goarch = architectureMap[process.arch];

  if (!goos || !goarch) {
    throw new Error(`unsupported platform or architecture: ${process.platform}/${process.arch}`);
  }

  const binaryName = goos === "windows" ? "skiller.exe" : "skiller";
  const archiveName = `skiller-npm_${packageJSON.version}_${goos}_${goarch}.tar.gz`;
  const url = `https://github.com/Kklyee/skiller/releases/download/v${packageJSON.version}/${archiveName}`;

  return {
    archiveName,
    binaryName,
    url
  };
}

async function install() {
  const target = targetForCurrentMachine();
  const binary = extractBinary(await download(target.url), target.binaryName);
  const binDirectory = path.join(packageRoot, "bin");
  const binaryPath = path.join(binDirectory, target.binaryName);

  fs.mkdirSync(binDirectory, { recursive: true });
  fs.writeFileSync(binaryPath, binary, { mode: 0o755 });
  if (process.platform !== "win32") {
    fs.chmodSync(binaryPath, 0o755);
  }

  console.log(`Installed Skiller ${packageJSON.version} from ${target.archiveName}`);
}

if (require.main === module) {
  install().catch((error) => {
    console.error(`Skiller installation failed: ${error.message}`);
    process.exitCode = 1;
  });
}

module.exports = {
  extractBinary,
  targetForCurrentMachine
};
