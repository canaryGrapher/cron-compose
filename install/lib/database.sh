#!/usr/bin/env bash
# Database provisioning and migration. Uses one of four strategies chosen during
# configure: install PostgreSQL via the OS package manager, an existing DSN (no-op),
# psql role/db creation in a running server, or a Docker Postgres.

# Bring the database into existence / make it reachable.
provision_database() {
  case "$DB_METHOD" in
    native)  provision_db_native ;;
    docker)  provision_db_docker ;;
    psql)    provision_db_psql ;;
    *)       step "Database"; ok "using existing Postgres at the supplied DATABASE_URL" ;;
  esac
}

# Run a command. In dry-run mode (CC_DB_DRY_RUN=1) just print it instead.
db_run() {
  if [ "${CC_DB_DRY_RUN:-0}" = "1" ]; then info "[dry-run] $*"; return 0; fi
  "$@"
}

# Prefix for commands that need root (package install, service control). Empty when
# already root or on macOS (brew runs as the user); "sudo" otherwise.
need_root() {
  { [ "$PLATFORM" = "macos" ] || [ "$(id -u)" -eq 0 ]; } && printf '' || printf 'sudo'
}

# Run a command as the cluster superuser OS user. On Linux that is the postgres user
# (via sudo, which also works when already root); on macOS the current brew user is a
# superuser so the command runs directly.
as_postgres() {
  if [ "$PLATFORM" = "macos" ]; then db_run "$@"; else db_run sudo -u postgres "$@"; fi
}

# Set SU_PSQL to the array used to run psql as the superuser, connected to the maintenance
# "postgres" database (works the same on Linux and macOS).
set_su_psql() {
  if [ "$PLATFORM" = "macos" ]; then SU_PSQL=(psql -d postgres)
  else SU_PSQL=(sudo -u postgres psql -d postgres); fi
}

# Install + start PostgreSQL with the detected package manager, then create the role and
# database. Idempotent: skips the package install if a server is already present.
provision_db_native() {
  step "Installing and configuring PostgreSQL ($PKG_MGR)"
  [ -n "${PKG_MGR:-}" ] || die "no package manager detected; choose an existing or Docker database instead"

  if command -v psql >/dev/null 2>&1 || command -v pg_ctl >/dev/null 2>&1; then
    ok "PostgreSQL already installed; skipping package install"
  else
    pg_install
  fi
  pg_ensure_running
  set_su_psql
  pg_wait_ready
  pg_create_role_and_db
  ok "PostgreSQL ready; database '$DB_NAME' owned by '$DB_USER'"
}

# Install the server package for each supported manager.
pg_install() {
  local R; R="$(need_root)"
  info "installing the PostgreSQL server (you may be prompted for your sudo password)..."
  case "$PKG_MGR" in
    apt-get) db_run $R apt-get update -qq && db_run $R apt-get install -y postgresql postgresql-client ;;
    dnf)     db_run $R dnf install -y postgresql-server postgresql ;;
    yum)     db_run $R yum install -y postgresql-server postgresql ;;
    zypper)  db_run $R zypper --non-interactive install postgresql-server postgresql ;;
    pacman)  db_run $R pacman -S --noconfirm postgresql ;;
    apk)     db_run $R apk add --no-cache postgresql postgresql-client ;;
    brew)    db_run brew install postgresql@16 ;;
    *)       die "don't know how to install PostgreSQL with '$PKG_MGR'; install it yourself and re-run choosing 'existing'" ;;
  esac
  ok "server package installed"
}

# Initialize a cluster where the package manager doesn't, then start + enable the service.
pg_ensure_running() {
  local R; R="$(need_root)"
  case "$PKG_MGR" in
    apt-get)
      # Debian/Ubuntu/Raspberry Pi OS auto-create a cluster and start it on install.
      db_run $R systemctl enable --now postgresql 2>/dev/null || db_run $R service postgresql start || true
      ;;
    dnf|yum)
      db_run $R postgresql-setup --initdb 2>/dev/null || true
      db_run $R systemctl enable --now postgresql
      ;;
    zypper)
      db_run $R systemctl enable --now postgresql
      ;;
    pacman)
      [ -d /var/lib/postgres/data ] || db_run $R install -d -o postgres -g postgres /var/lib/postgres/data
      as_postgres initdb -D /var/lib/postgres/data 2>/dev/null || true
      db_run $R systemctl enable --now postgresql
      ;;
    apk)
      as_postgres initdb -D /var/lib/postgresql/data 2>/dev/null || true
      db_run $R rc-update add postgresql 2>/dev/null || true
      db_run $R rc-service postgresql start || true
      ;;
    brew)
      db_run brew services start postgresql@16
      ;;
  esac
  ok "service started"
}

# Block until the server accepts connections (best effort, ~30s).
pg_wait_ready() {
  [ "${CC_DB_DRY_RUN:-0}" = "1" ] && return 0
  local i=0
  while [ "$i" -lt 30 ]; do
    "${SU_PSQL[@]}" -tAc "select 1" >/dev/null 2>&1 && { ok "server is accepting connections"; return 0; }
    i=$((i + 1)); sleep 1
  done
  warn "server did not report ready in time; continuing (migrations will retry)"
}

# Create the login role (with password, for TCP auth) and an owned database. Idempotent.
pg_create_role_and_db() {
  info "creating role '$DB_USER' and database '$DB_NAME'..."
  if [ "${CC_DB_DRY_RUN:-0}" = "1" ]; then
    info "[dry-run] ${SU_PSQL[*]} -c \"create role \\\"$DB_USER\\\" login password '****'\""
    info "[dry-run] ${SU_PSQL[*]} -c \"create database \\\"$DB_NAME\\\" owner \\\"$DB_USER\\\"\""
    return 0
  fi
  if [ "$("${SU_PSQL[@]}" -tAc "select 1 from pg_roles where rolname='$DB_USER'" 2>/dev/null)" != "1" ]; then
    "${SU_PSQL[@]}" -v ON_ERROR_STOP=1 -c "create role \"$DB_USER\" login password '$DB_PASS'" >/dev/null \
      && ok "created role $DB_USER" || die "failed to create role $DB_USER"
  else
    "${SU_PSQL[@]}" -c "alter role \"$DB_USER\" login password '$DB_PASS'" >/dev/null 2>&1
    ok "role $DB_USER already existed (password updated)"
  fi
  if [ "$("${SU_PSQL[@]}" -tAc "select 1 from pg_database where datname='$DB_NAME'" 2>/dev/null)" != "1" ]; then
    "${SU_PSQL[@]}" -v ON_ERROR_STOP=1 -c "create database \"$DB_NAME\" owner \"$DB_USER\"" >/dev/null \
      && ok "created database $DB_NAME" || die "failed to create database $DB_NAME"
  else
    ok "database $DB_NAME already existed"
  fi
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

  # Hold the superuser password in memory (PGPASSWORD) for the whole session so the
  # individual psql commands below never re-prompt. Seed it from the configure step;
  # every psql call uses -w (never prompt) and relies on this instead.
  [ -n "${DB_SUPER_PASS:-}" ] && export PGPASSWORD="$DB_SUPER_PASS"

  # Probe connectivity without prompting. If the server wants a password we don't have
  # yet (left blank at configure, but the server isn't trust/peer), ask once here and
  # reuse it for every command that follows.
  if ! psql -w "$super_url" -tAc "select 1" >/dev/null 2>&1; then
    if [ "${NONINTERACTIVE:-0}" != "1" ]; then
      local tries=0 supw=""
      while [ "$tries" -lt 3 ]; do
        printf '%sPassword for superuser %s%s: ' "$C_BOLD" "$DB_SUPERUSER" "$C_RESET" >/dev/tty
        IFS= read -rs supw </dev/tty || supw=""
        printf '\n' >/dev/tty
        export PGPASSWORD="$supw"
        if psql -w "$super_url" -tAc "select 1" >/dev/null 2>&1; then break; fi
        warn "could not connect as '$DB_SUPERUSER'; check the password and try again"
        tries=$((tries + 1))
      done
    fi
    if ! psql -w "$super_url" -tAc "select 1" >/dev/null 2>&1; then
      unset PGPASSWORD
      die "could not connect as superuser '$DB_SUPERUSER' to $DB_HOST:$DB_PORT. Create the database manually and re-run choosing 'existing'."
    fi
  fi
  ok "connected as superuser $DB_SUPERUSER"

  # Role: create if missing, then ensure the password matches.
  if [ "$(psql -w "$super_url" -tAc "select 1 from pg_roles where rolname='$DB_USER'")" != "1" ]; then
    psql -w "$super_url" -v ON_ERROR_STOP=1 -c "create role \"$DB_USER\" login password '$DB_PASS'" >/dev/null \
      && ok "created role $DB_USER" || die "failed to create role $DB_USER"
  else
    psql -w "$super_url" -c "alter role \"$DB_USER\" login password '$DB_PASS'" >/dev/null 2>&1
    ok "role $DB_USER already existed (password updated)"
  fi

  # Database: create if missing, owned by the role.
  if [ "$(psql -w "$super_url" -tAc "select 1 from pg_database where datname='$DB_NAME'")" != "1" ]; then
    psql -w "$super_url" -v ON_ERROR_STOP=1 -c "create database \"$DB_NAME\" owner \"$DB_USER\"" >/dev/null \
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
  # Run directly (no pipe) so a failure is caught by the if instead of aborting under
  # `set -o pipefail`. The tool prints its own indented apply/skip lines.
  if ! DATABASE_URL="$DATABASE_URL" "$migrate_bin" -dir "$REPO_ROOT/migrations"; then
    die "migrations failed (is DATABASE_URL reachable and can the role create tables?)"
  fi
  ok "schema is up to date"
}
