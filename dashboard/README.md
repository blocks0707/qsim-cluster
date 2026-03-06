# QSim Cluster Dashboard

A modern web dashboard for managing and monitoring quantum simulation clusters built with [qsim-cluster](https://github.com/blocks0707/qsim-cluster).

![Dashboard Screenshot](docs/screenshot-placeholder.png)

## Features

| Page | Description |
|------|-------------|
| **Login** | Connect to any qsim-cluster API with endpoint URL + Bearer token |
| **Overview** | Cluster status, resource gauges, recent jobs, quick stats |
| **Jobs** | List, create, cancel, retry jobs; view results & logs |
| **Nodes** | Node grid with status, qubits, utilization |
| **Jupyter** | Manage Jupyter notebook sessions on the cluster |
| **Metrics** | Qubit distribution, job throughput, complexity charts, resource trends |

## Getting Started

```bash
cd dashboard
npm install
npm run dev
```

Open [http://localhost:3000](http://localhost:3000). You'll be redirected to the login page.

### Environment Variables (optional)

| Variable | Default | Description |
|----------|---------|-------------|
| `NEXT_PUBLIC_API_URL` | `http://localhost:8080` | Fallback API endpoint (overridden by login) |

## Tech Stack

- **Next.js 15** (App Router)
- **Tailwind CSS** — dark theme
- **Recharts** — metrics visualization
- **Lucide React** — icons
- **TypeScript** — full type safety

## Authentication

The dashboard stores the API URL and Bearer token in `localStorage`. On every API call, the token is sent as `Authorization: Bearer <token>`. A 401 response automatically redirects to the login page.

## Responsive Design

- Desktop: fixed sidebar navigation
- Mobile: hamburger menu with slide-over sidebar overlay

## License

MIT
