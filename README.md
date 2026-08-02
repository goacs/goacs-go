# GoACS

GoACS is an Auto Configuration Server (ACS) implementing the TR-069/CWMP protocol for remote management of customer-premises equipment (routers, ONTs, and similar devices). It includes a REST API and a Vue 3 admin panel for managing devices, parameters, provisioning rules, firmware, and users.

This is a Go rewrite of an original Laravel/Vue 2 implementation, with its own REST API contract (see [AGENTS.md](AGENTS.md) for details — it is not wire-compatible with the original).

## Features

- TR-069/CWMP session handling: Inform, GetParameterValues/Names, SetParameterValues, AddObject/DeleteObject, Reboot, FactoryReset, Download, transfer complete, faults
- Task queue per device (reboot, factory reset, firmware upload, add/delete object, run script) and global tasks
- Provisioning rules engine: match CWMP events/requests/parameter conditions, run Lua scripts against the device automatically
- Parameter templates, assignable to devices with priority
- On-demand parameter lookup and forced re-provisioning via Connection Request
- Firmware/file storage with upload/download
- Live device log streaming over Socket.IO, plus paginated log/fault history
- JWT-authenticated REST API and a Vue 3 admin panel (PrimeVue, Pinia)

## Architecture

```
goacs-go/
├── acs/          TR-069/CWMP protocol engine
├── http/         Gin REST API + Socket.IO
├── repository/   MySQL/MariaDB data access
├── models/       Domain types
├── contrib/      SQL migrations + DB config
└── frontend/     Vue 3 admin panel (separate app, own deployment)
```

The backend and frontend are deployed independently (separate processes/origins) and talk over HTTP + CORS + Socket.IO. See [AGENTS.md](AGENTS.md) for a full tour of the codebase, conventions, and known gotchas.

## Prerequisites

- Go 1.23+
- Node.js 20+ and npm
- Docker (for the bundled MariaDB), or your own MySQL/MariaDB 10.4+ instance

## Quick start

**Backend:**

```bash
cp .env.example .env
# edit .env if you need to change ports/credentials — defaults work out of the box
docker compose up -d goacs-db     # starts MariaDB and applies contrib/database/*.sql on first run
go run main.go                    # ACS + API server on :8085
```

**Frontend:**

```bash
cd frontend
npm install
npm run dev                       # dev server on :5173
```

Open `http://localhost:5173` and log in with the seeded `admin` / `admin` account.

## Configuration

Key environment variables (see `.env.example` for the full list):

| Variable | Purpose |
|---|---|
| `HTTP_PORT` | Port for both the CWMP (`/acs`) and REST (`/api`) endpoints |
| `JWT_SECRET` | Signing secret for API auth tokens — change this for anything beyond local dev |
| `CORS_ALLOWED_ORIGINS` | Comma-separated origins allowed to call the API / connect over Socket.IO (must include the frontend's dev/prod origin) |
| `MYSQL_*` | Database connection |
| `FILESTORE_PATH` | Where uploaded firmware/files are stored |

## Testing

```bash
go build ./...
go test ./...
```

```bash
cd frontend
npx vue-tsc -b --noEmit
npm run build
```

Manual API testing: `.http` request files are in `testrequests/` (compatible with the VS Code REST Client extension or JetBrains' HTTP Client).

## Contributing

See [AGENTS.md](AGENTS.md) for codebase conventions, the REST response envelope shape, and a list of known gotchas worth reading before you change the ACS engine or the repository layer.

## Sponsors

##### [MULTIPLAY](https://multiplay.pl)
![GRUPA MULTIPLAY](.github/sponsors/mpl_logo.png "GRUPA MULTIPLAY")
