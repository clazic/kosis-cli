#!/usr/bin/env node
"use strict";

const https = require("https");
const http = require("http");
const fs = require("fs");
const path = require("path");
const os = require("os");
const { execSync } = require("child_process");

const VERSION = require("./package.json").version;
const REPO = "clazic/kosis-cli";

function getPlatformInfo() {
  const platform = os.platform();
  const arch = os.arch();

  const platformMap = {
    darwin: "darwin",
    linux: "linux",
    win32: "windows",
  };

  const archMap = {
    x64: "amd64",
    arm64: "arm64",
  };

  const goos = platformMap[platform];
  const goarch = archMap[arch];

  if (!goos || !goarch) {
    throw new Error(
      `지원하지 않는 플랫폼: ${platform}/${arch}\n` +
        "지원 플랫폼: darwin/arm64, darwin/amd64, linux/amd64, linux/arm64, windows/amd64"
    );
  }

  const ext = platform === "win32" ? ".exe" : "";
  const artifact = `kosis-${goos}-${goarch}${ext}`;

  return { goos, goarch, artifact, ext };
}

function download(url) {
  return new Promise((resolve, reject) => {
    const client = url.startsWith("https") ? https : http;
    client
      .get(url, { headers: { "User-Agent": "kosis-npm-installer" } }, (res) => {
        // Follow redirects (GitHub releases redirect)
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          return download(res.headers.location).then(resolve).catch(reject);
        }
        if (res.statusCode !== 200) {
          return reject(new Error(`다운로드 실패: HTTP ${res.statusCode} for ${url}`));
        }
        const chunks = [];
        res.on("data", (chunk) => chunks.push(chunk));
        res.on("end", () => resolve(Buffer.concat(chunks)));
        res.on("error", reject);
      })
      .on("error", reject);
  });
}

async function main() {
  const { artifact, ext } = getPlatformInfo();
  const binDir = path.join(__dirname, "bin");
  const binPath = path.join(binDir, `kosis${ext}`);

  // Skip if binary already exists and is correct version
  if (fs.existsSync(binPath)) {
    try {
      const output = execSync(`"${binPath}" --version`, { encoding: "utf8", timeout: 5000 });
      if (output.includes(VERSION) || output.includes("v" + VERSION)) {
        console.log(`kosis v${VERSION} 이미 설치됨`);
        return;
      }
    } catch {
      // Version mismatch or error, re-download
    }
  }

  const tag = `v${VERSION}`;
  const url = `https://github.com/${REPO}/releases/download/${tag}/${artifact}`;

  console.log(`kosis v${VERSION} 다운로드 중... (${artifact})`);

  try {
    const data = await download(url);

    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }

    fs.writeFileSync(binPath, data);
    fs.chmodSync(binPath, 0o755);

    console.log(`kosis v${VERSION} 설치 완료!`);
  } catch (err) {
    console.error(`kosis 설치 실패: ${err.message}`);
    console.error(`수동 다운로드: https://github.com/${REPO}/releases/tag/${tag}`);
    process.exit(1);
  }
}

main();
