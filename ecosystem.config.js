module.exports = {
  apps: [{
    name: 'parameter-store',
    script: './parameter-store',
    args: '-port 8847 -data data.jsonl'
  }]
};
