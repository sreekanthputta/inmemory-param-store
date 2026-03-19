const fs = require('fs');
const path = require('path');

// Load .env file if it exists
const envPath = path.join(__dirname, '.env');
const env = {};
if (fs.existsSync(envPath)) {
  fs.readFileSync(envPath, 'utf8').split('\n').forEach(line => {
    const [key, ...val] = line.split('=');
    if (key && val.length) env[key.trim()] = val.join('=').trim();
  });
}

module.exports = {
  apps: [{
    name: 'parameter-store',
    script: './parameter-store',
    args: '-port 8847 -data data.jsonl',
    env: env
  }]
};
