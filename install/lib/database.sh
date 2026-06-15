#!/usr/bin/env bash
# Database provisioning and migration. Uses one of three strategies chosen during
# configure: an existing DSN (no-op), psql role/db creation, or a Docker Postgres.

# Bring the database into existence / make it reachable.
provision_database() {
  case "$DB_METHOD" in
    docker)  provision_db_docker ;;
    psql)    provision_db_psql ;;
    *)       step "Database"; ok "using existing Postgres at the supplied DATABASE_URL" ;;
  esac
}

provision_db_docker() {
  step "Starting Postgres in Docker"
  local compose
  if docker compose version >/dev/null 2>&1; then compose="docker compose"
  elif command -v docker-compose >/dev/null 2>&1; then compose="docker-compose"
  else die "docker compose not available"; fi
  ( cd "$REPO_ROOT" && $compose -f docker-compose.yml up -d postgres ) || die "failed to start Postgres container"
  info "waiting for Postgres to accept connections..."
  local i=0
  while [ "$i" -lt 60 ]; do
    if ( cd "$REPO_ROOT" && $compose -f docker-compose.yml exec -T postgres pg_isready -U "$DB_USER" >/dev/null 2>&1 ); then
      ok "Postgres is ready on port $DB_PORT"; return 0
    fi
    i=$((i + 1)); sleep 1
  done
  die "Postgres did not become ready in time"
}

provision_db_psql() {
  step "Creating database role and schema owner via psql"
  local super_url="postgres://$DB_SUPERUSER@$DB_HOST:$DB_PORT/postgres?sslmode=disable"
  [ -n "${DB_SUPER_PASS:-}" ] && export PGPASSWORD="$DB_SUPER_PASS"

  if ! psql "$super_url" -tAc "select 1" >/dev/null 2>&1; then
    unset PGPASSWORD
    die "could not connect as superuser '$DB_SUPERUSER' to $DB_HOST:$DB_PORT. Create the database manually and re-run choosing 'existing'."
  fi

  # Role: create if missing, then ensure the password matches.
  if [ "$(psql "$super_url" -tAc "select 1 from pg_roles where rolname='$DB_USER'")" != "1" ]; then
    psql "$super_url" -v ON_ERROR_STOP=1 -c "create role \"$DB_USER\" login password '$DB_PASS'" >/dev/null \
      && ok "created role $DB_USER" || die "failed to create role $DB_USER"
  else
    psql "$super_url" -c "alter role \"$DB_USER\" login password '$DB_PASS'" >/dev/null 2>&1
    ok "role $DB_USER already existed (password updated)"
  fi

  # Database: create if missing, owned by the role.
  if [ "$(psql "$super_url" -tAc "select 1 from pg_database where datname='$DB_NAME'")" != "1" ]; then
    psql "$super_url" -v ON_ERROR_STOP=1 -c "create database \"$DB_NAME\" owner \"$DB_USER\"" >/dev/null \
      && ok "created database $DB_NAME" || die "failed to create database $DB_NAME"
  else
    ok "database $DB_NAME already existed"
  fi
  unset PGPASSWORD
}

apply_migrations() {
  step "Applying database migrations"
  local migrate_bin="$REPO_ROOT/control-plane/bin/migrate"
  [ -x "$migrate_bin" ] || die "migrate tool not built (expected $migrate_bin)"
  DATABASE_URL="$DATABASE_URL" "$migrate_bin" -dir "$REPO_ROOT/migrations" 2>&1 | while IFS= read -r l; do info "$l"; done
  # Propagate the migrate tool's exit status, not the while loop's.
  local rc="${PIPESTATUS[0]}"
  [ "$rc" = "0" ] || die "migrations failed (exit $rc)"
  ok "schema is up to date"
}
