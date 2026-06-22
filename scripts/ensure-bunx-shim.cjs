const fs = require('node:fs');
const path = require('node:path');

const binDir = path.join(__dirname, '..', 'node_modules', '.bin');
const shimPath = path.join(binDir, 'bunx.cmd');

const content = '@echo off\r\nbun x %*\r\n';

fs.mkdirSync(binDir, { recursive: true });
fs.writeFileSync(shimPath, content, 'utf8');