const https = require("https");
const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");
const os = require("os");

const VERSION = require("./package.json").version;
const REPO = "KinjiKawaguchi/sessionctl";

function getPlatform() {
  const platform = os.platform();
  const arch = os.arch();

  const osMap = { darwin: "darwin", linux: "linux", win32: "windows" };
  const archMap = { x64: "amd64", arm64: "arm64" };

  const goOS = osMap[platform];
  const goArch = archMap[arch];

  if (!goOS || !goArch) {
    throw new Error(`Unsupported platform: ${platform}/${arch}`);
  }

  return { goOS, goArch };
}

function download(url) {
  return new Promise((resolve, reject) => {
    https
      .get(url, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          return download(res.headers.location).then(resolve).catch(reject);
        }
        if (res.statusCode !== 200) {
          return reject(new Error(`HTTP ${res.statusCode}: ${url}`));
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
  const { goOS, goArch } = getPlatform();
  const ext = goOS === "windows" ? "zip" : "tar.gz";
  const archiveName = `sessionctl_${goOS}_${goArch}.${ext}`;
  const url = `https://github.com/${REPO}/releases/download/v${VERSION}/${archiveName}`;

  console.log(`Downloading sessionctl v${VERSION} for ${goOS}/${goArch}...`);

  const data = await download(url);
  const tmpFile = path.join(os.tmpdir(), archiveName);
  fs.writeFileSync(tmpFile, data);

  const binDir = path.join(__dirname, "bin");
  fs.mkdirSync(binDir, { recursive: true });

  if (ext === "tar.gz") {
    execSync(`tar xzf "${tmpFile}" -C "${binDir}" sessionctl`, { stdio: "inherit" });
    fs.chmodSync(path.join(binDir, "sessionctl"), 0o755);
  } else {
    execSync(`unzip -o "${tmpFile}" sessionctl.exe -d "${binDir}"`, { stdio: "inherit" });
  }

  fs.unlinkSync(tmpFile);
  console.log("sessionctl installed successfully.");
}

main().catch((err) => {
  console.error("Failed to install sessionctl:", err.message);
  process.exit(1);
});
