# Database Migrations

## How it works

Migrations run automatically via an init container in the API Server deployment.

1. Migration SQL files are stored as a ConfigMap (`db-migrations`)
2. API Server pod has an init container (`db-migrate`) using `postgres:15` image
3. Init container waits for PostgreSQL readiness, then runs all migration files
4. API Server main container starts only after migrations complete

## Adding new migrations

1. Create a new file: `deploy/migrations/002_<name>.sql`
2. Update the ConfigMap:
   ```bash
   kubectl create configmap db-migrations \
     --from-file=001_init.sql=deploy/migrations/001_init.sql \
     --from-file=002_<name>.sql=deploy/migrations/002_<name>.sql \
     -n quantum-system --dry-run=client -o yaml | kubectl apply -f -
   ```
3. Update the init container command to include the new file
4. Restart the API Server: `kubectl rollout restart deployment/api-server -n quantum-system`

## Files

- `001_init.sql` — Initial schema (quantum_jobs table + indexes)
