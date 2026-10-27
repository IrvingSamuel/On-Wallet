module.exports = {
  apps: [
    {
      name: "carteira",
      cwd: __dirname,
      script: "./backend/bin/carteira",
      interpreter: "none",
      autorestart: true,
      max_restarts: 10,
    },
  ],
};
