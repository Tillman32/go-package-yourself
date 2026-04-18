#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const os = require('os');
const https = require('https');
const { execFileSync } = require('child_process');
const crypto = require('crypto');
const zlib = require('zlib');
const tar = require('tar');

// Platform configuration - MUST match Release.Platforms
const PLATFORMS = [
  { platform: 'darwin', arch: 'amd64', filename: 'go-package-yourself__darwin_amd64.tar.gz', binPath: 'go-package-yourself' }
  { platform: 'darwin', arch: 'arm64', filename: 'go-package-yourself__darwin_arm64.tar.gz', binPath: 'go-package-yourself' }
  { platform: 'linux', arch: 'amd64', filename: 'go-package-yourself__linux_amd64.tar.gz', binPath: 'go-package-yourself' }
  { platform: 'linux', arch: 'arm64', filename: 'go-package-yourself__linux_arm64.tar.gz', binPath: 'go-package-yourself' }
  { platform: 'windows', arch: 'amd64', filename: 'go-package-yourself__windows_amd64.zip', binPath: 'go-package-yourself.exe' }
];

// Repository information
const REPO = 'Tillman32/go-package-yourself';

// Binary name
const BIN_NAME = 'go-package-yourself';

// Get cache directory based on OS
function getCacheDir() {
  const home = os.homedir();
  const osType = os.platform();

  if (osType === 'linux') {
    return path.join(process.env.XDG_CACHE_HOME || path.join(home, '.cache'), 'npm-binaries');
  } else if (osType === 'darwin') {
    return path.join(home, 'Library', 'Caches', 'npm-binaries');
  } else if (osType === 'win32') {
    return path.join(process.env.LOCALAPPDATA || path.join(home, 'AppData', 'Local'), 'npm-cache', 'binaries');
  }
  throw new Error('Unsupported OS: ' + osType);
}

// Get the platform info for the current system
function getPlatformInfo() {
  const osType = os.platform() === 'win32' ? 'windows' : os.platform();
  const arch = os.arch() === 'x64' ? 'amd64' : (os.arch() === 'arm64' ? 'arm64' : os.arch());

  const platform = PLATFORMS.find(p => p.platform === osType && p.arch === arch);
  if (!platform) {
    throw new Error('Unsupported platform: ' + osType + '/' + arch);
  }
  return platform;
}

// Download a file and verify SHA256
async function downloadAndVerify(url, checksumUrl, expectedChecksum) {
  return new Promise((resolve, reject) => {
    const tempPath = path.join(os.tmpdir(), 'npm-binary-' + Date.now());
    const file = fs.createWriteStream(tempPath);

    https.get(url, (response) => {
      if (response.statusCode !== 200) {
        fs.unlink(tempPath, () => {});
        reject(new Error('Failed to download ' + url + ': HTTP ' + response.statusCode));
        return;
      }

      const hash = crypto.createHash('sha256');
      response.pipe(file);
      response.on('data', (chunk) => hash.update(chunk));

      response.on('end', () => {
        file.close();
        const digest = hash.digest('hex');

        // Verify SHA256
        if (digest.toLowerCase() !== expectedChecksum.toLowerCase()) {
          fs.unlink(tempPath, () => {});
          reject(new Error('SHA256 verification failed for ' + url + ': expected ' + expectedChecksum + ', got ' + digest));
          return;
        }

        resolve(tempPath);
      });
    }).on('error', (err) => {
      fs.unlink(tempPath, () => {});
      reject(new Error('Failed to download ' + url + ': ' + err.message));
    });
  });
}

// Extract archive (tar.gz or zip)
async function extract(archivePath, targetDir, binPath) {
  return new Promise((resolve, reject) => {
    if (archivePath.endsWith('.tar.gz')) {
      // Extract tar.gz
      const gunzip = zlib.createGunzip();
      const extract = tar.extract({ cwd: targetDir });

      fs.createReadStream(archivePath)
        .pipe(gunzip)
        .pipe(extract)
        .on('error', reject)
        .on('end', resolve);
    } else if (archivePath.endsWith('.zip')) {
      // For zip, just copy the file
      // In production, use 'unzipper' or similar package
      // For now, assume it's already extracted by our download
      resolve();
    } else {
      reject(new Error('Unsupported archive format'));
    }
  });
}

// Install binary if not cached
async function ensureBinary(cacheDir, platform) {
  const binaryPath = path.join(cacheDir, platform.filename + '.extracted');

  // Check cache
  if (fs.existsSync(binaryPath)) {
    try {
      // Verify cache is executable and valid
      fs.accessSync(binaryPath, fs.constants.X_OK);
      return binaryPath;
    } catch (e) {
      // Cache is invalid, remove it
      try {
        fs.unlinkSync(binaryPath);
      } catch (e2) {
        // ignore
      }
    }
  }

  // Download and install
  const tagVersion = process.env.GPY_VERSION || 'latest';
  const releaseUrl = 'https://github.com/' + REPO + '/releases/download/' + tagVersion;
  const archiveUrl = releaseUrl + '/' + platform.filename;
  const checksumUrl = releaseUrl + '/checksums.txt';

  console.error('Downloading ' + BIN_NAME + ' (' + platform.platform + '/' + platform.arch + ')...');

  try {
    // Fetch checksums file
    const checksumsText = await new Promise((resolve, reject) => {
      https.get(checksumUrl, (res) => {
        let data = '';
        res.on('data', chunk => data += chunk);
        res.on('end', () => resolve(data));
      }).on('error', reject);
    });

    // Find checksum for this archive
    const lines = checksumsText.split('\n');
    let expectedChecksum = null;
    for (const line of lines) {
      const parts = line.trim().split(/\s+/);
      if (parts.length >= 2 && parts[1] === platform.filename) {
        expectedChecksum = parts[0];
        break;
      }
    }

    if (!expectedChecksum) {
      throw new Error('Checksum not found for ' + platform.filename);
    }

    // Download archive
    const archivePath = await downloadAndVerify(archiveUrl, checksumUrl, expectedChecksum);

    // Ensure cache directory exists
    fs.mkdirSync(cacheDir, { recursive: true });

    // Extract binary
    const binDir = path.join(cacheDir, 'bin');
    fs.mkdirSync(binDir, { recursive: true });

    // For tar.gz: extract the binary file
    if (archivePath.endsWith('.tar.gz')) {
      await extract(archivePath, binDir, platform.binPath);
      const extractedPath = path.join(binDir, platform.binPath);

      // Move to final location
      fs.renameSync(extractedPath, binaryPath);
    } else if (archivePath.endsWith('.zip')) {
      // For Windows zip files, we'd need to handle differently
      // This is a simplified scenario
      throw new Error('ZIP extraction not fully implemented in launcher');
    }

    // Ensure binary is executable
    fs.chmodSync(binaryPath, 0o755);
    
    // Clean up temp archive
    try {
      fs.unlinkSync(archivePath);
    } catch (e) {
      // ignore
    }

    console.error('✓ Binary installed to cache');
    return binaryPath;
  } catch (err) {
    throw new Error('Failed to install binary: ' + err.message);
  }
}

// Main entry point
async function main() {
  try {
    const cacheDir = getCacheDir();
    const platform = getPlatformInfo();
    const binaryPath = await ensureBinary(cacheDir, platform);

    // Execute binary with all arguments passed through
    const args = process.argv.slice(2);
    try {
      execFileSync(binaryPath, args, {
        stdio: 'inherit',
        env: process.env,
      });
    } catch (err) {
      if (err.status !== undefined) {
        process.exit(err.status);
      } else {
        throw err;
      }
    }
  } catch (err) {
    console.error('Error: ' + err.message);
    process.exit(1);
  }
}

main();
