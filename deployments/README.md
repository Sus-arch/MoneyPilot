# Dockerfile for MoneyPilot API

This folder contains a multi-stage `Dockerfile` to build the Go API (located under `cmd/api`) and produce a small runtime image based on Alpine.

Notes and recommendations
- The image is built with Go 1.24 (alpine) and the final image is Alpine 3.18.
- Do NOT copy your `.env` into the image. Pass secrets and runtime configuration via `--env`, `--env-file` or your `docker-compose.yml`.
- The binary is built with static flags (CGO disabled). If you need cgo, adjust the builder and final image accordingly.

Quick build & run (PowerShell):

```powershell
# build the image
docker build -t moneypilot-api -f deployments/Dockerfile .

# run the container (expose port and load env from file in working dir)
docker run -p 8080:8080 --env-file .\.env --name moneypilot-api moneypilot-api
```

If your app reads port from `SERVER_PORT` or similar env var, set it with `-e "SERVER_PORT=8080"` or include it in your `.env` file.

Optional: integrate into `deployments/docker-compose.yml` by mounting `.env` or defining environment entries.
