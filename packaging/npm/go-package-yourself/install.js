#!/usr/bin/env node

// This script runs after npm install to download the binary.
// In a real implementation, this would handle post-install setup.
// For now, we rely on the main index.js to download on first run.

const fs = require('fs');
const path = require('path');

// Create bin directory structure
const binDir = path.join(__dirname, 'bin');
if (!fs.existsSync(binDir)) {
  fs.mkdirSync(binDir, { recursive: true });
}

console.log('✓ npm package installed successfully');
