# Pixel Present

> Un regalo pequeño, hecho píxel a píxel, que se descubre como una sorpresa.

**Pixel Present** es una aplicación web para crear **regalos digitales interactivos** a partir de ilustraciones pixel art. Una persona dibuja píxel a píxel, añade un mensaje personal y elige una animación de revelación; la destinataria recibe una **URL privada** donde descubre el regalo tras una pequeña interacción (en el MVP, una explosión de confeti que ensambla el dibujo) y puede dejar una reacción.

🌐 **Producción:** [https://pixelpresent.eu](https://pixelpresent.eu)

---

## Cómo funciona

**Quien crea el regalo:**
1. Entra y se autentica con un **magic link** (enlace de un solo uso enviado por email; no hay contraseñas).
2. Dibuja en el editor de pixel art (lienzo 8×8 – 128×128, lápiz, borrador, relleno, deshacer/rehacer, zoom).
3. Añade título y mensaje, elige la mecánica de revelación y **publica**.
4. Comparte la URL privada (`/g/{token}`) por el canal que quiera.

**Quien lo recibe:**
1. Abre el enlace y ve una pantalla de expectativa.
2. Interactúa (toca el confeti) → los píxeles se ensamblan → se revela el dibujo y el mensaje.
3. Puede reaccionar (emoji o texto); la reacción aparece en el dashboard del creador.

El detalle de producto está en [`pixel_present.md`](pixel_present.md); la arquitectura en [`pixel_present_arquitectura.md`](pixel_present_arquitectura.md); el backlog en [`pixel_present_tareas.md`](pixel_present_tareas.md).

---

## Stack

| Capa | Tecnología |
|---|---|
| Backend | **Go 1.26**, router [`chi`](https://github.com/go-chi/chi), CLI con `cobra`, logs con `zerolog` |
| Datos | **PostgreSQL 16** vía [`pgx/v5`](https://github.com/jackc/pgx) (`pgxpool`); queries tipadas con [`sqlc`](https://sqlc.dev); migraciones con [`golang-migrate`](https://github.com/golang-migrate/migrate) |
| Frontend | **Astro 7** (build estático), islas [**Preact**](https://preactjs.com/), **Tailwind 4**; el canvas (editor + animaciones) en TS vanilla |
| Empaquetado | El `dist/` de Astro se **embebe en el binario Go** (`go:embed`): un único proceso sirve estáticos + API |
| Email (MVP) | SMTP estándar — [Mailpit](https://github.com/axllent/mailpit) en dev, Proton en producción |

Principio de diseño: toda dependencia externa (BD, storage, email) se accede tras **interfaces Go**, para poder sustituir la implementación sin tocar la lógica de negocio.

---

## Estructura del repositorio

```text
cmd/server/          Punto de entrada (subcomandos: serve, migrate up/down)
internal/
  api/               Handlers HTTP y router (JSON API + servido de estáticos)
  auth/              Magic link, sesión firmada (HMAC), plantilla de email
  gifts/             Dominio de regalos (creación, visibilidad, view token)
  reactions/         Dominio de reacciones
  domain/            Tipos de dominio
  repository/        Implementación sobre Postgres
    db/migrations/   Migraciones SQL (embebidas)
    db/queries/      Queries fuente de sqlc
    db/sqlc/         Código generado por sqlc (no editar a mano)
  email/             Cliente SMTP
  web/               go:embed del dist de Astro
  util/              Carga de config y logger
web/                 Frontend Astro (islas Preact, canvas, estilos)
config.yaml          Plantilla de configuración (valores por env; sin secretos)
Taskfile.yml         Tareas de dev/build/test
```

---

## Desarrollo

### Requisitos

| Herramienta | Para qué | Nota |
|---|---|---|
| **Go** ≥ 1.26 | compilar/ejecutar el backend | |
| **Node** ≥ 22.12 + **pnpm** | frontend Astro | `corepack enable` trae pnpm |
| **Podman** + `podman compose` | Postgres y Mailpit en local | ver [`docker-compose.dev.yml`](docker-compose.dev.yml) |
| **[Task](https://taskfile.dev)** (`go-task`) | atajos del `Taskfile.yml` | opcional pero recomendado |
| **[sqlc](https://sqlc.dev)** | regenerar código de queries | solo si tocas SQL |

> `golang-migrate` **no** hace falta instalarlo aparte: las migraciones se aplican con el propio binario (`server migrate up`).

### Arranque (dos terminales)

```bash
# Terminal 1 — backend: levanta Postgres + Mailpit, migra y sirve la API en :8080
task dev-server

# Terminal 2 — frontend: servidor de Astro en :4321 (proxea /api → :8080)
task dev-front
```

Abre **http://localhost:4321**. Como el login es por magic link, el correo **no sale de tu máquina**: lo captura Mailpit — ábrelo en **http://localhost:8025** y pulsa el enlace para entrar.

Puertos en dev:

| Servicio | URL / puerto |
|---|---|
| Frontend (Astro dev) | http://localhost:4321 |
| API (Go) | http://localhost:8080 |
| Mailpit (UI de correos) | http://localhost:8025 |
| Postgres | `localhost:5432` |

Para parar los contenedores: `task dev-down`.

### Comandos (Taskfile)

| Comando | Acción |
|---|---|
| `task dev-server` | Contenedores + migraciones + API Go en primer plano |
| `task dev-front` | Servidor de desarrollo de Astro (proxy a la API) |
| `task dev-up` / `task dev-down` | Solo levantar/parar Postgres + Mailpit |
| `task migrate-up` / `task migrate-down` | Aplicar / revertir migraciones |
| `task sqlc-generate` | Regenerar el código Go a partir de las queries SQL |
| `task test` | `go test ./...` |
| `task build` | Build de producción (frontend + binario estático) → `bin/pixelpresent` |
| `task prod` | Compila y ejecuta como en producción (frontend embebido, API en :8080) |

### Rutas

| Ruta | Qué es |
|---|---|
| `/` | Landing |
| `/login` | Pedir magic link |
| `/dashboard` | Regalos del creador (requiere sesión) |
| `/editor` (`?id=<uuid>`) | Editor: sin `id` = nuevo, con `id` = editar |
| `/g/{token}` | Vista pública de revelación (destinataria) |
| `/api/*` | API JSON |

### Documentación de la API

La API está descrita en **OpenAPI 3.1** en [`docs/openapi.yaml`](docs/openapi.yaml) (la fuente de verdad). En desarrollo, el binario sirve además una UI interactiva (Scalar):

| Recurso | URL (dev) |
|---|---|
| UI interactiva | http://localhost:4321/api/docs |
| Spec en crudo | http://localhost:4321/api/docs/openapi.yaml |

> Las rutas de docs son **solo de desarrollo**: no se montan cuando `environment=production`, así que en producción no existen (cero superficie de ataque). Si tocas la API, actualiza `docs/openapi.yaml` en el mismo commit.

---

## Configuración

`config.yaml` es una **plantilla versionada sin secretos**: los valores se inyectan por variables de entorno, que el loader expande (`${VAR}` / `${VAR:-default}`). En dev, los defaults ya apuntan al Postgres y Mailpit de `docker-compose.dev.yml`, así que no necesitas configurar nada. Las claves disponibles están documentadas en [`.env.example`](.env.example); en producción las pone systemd.

---

## Tests

```bash
task test
```

- **Dominio**: contra *fakes* en memoria de las interfaces, sin red ni BD.
- **Repositorio**: contra una base `pixelpresent_test` en el Postgres de dev, cada test en una transacción que se revierte.

---

## Producción

Build → binario Go estático con el frontend embebido → se ejecuta con **systemd** detrás de **Caddy** (TLS automático + reverse proxy). Postgres nativo en el VPS (socket unix, esquema y roles dedicados). El binario se genera con:

```bash
./scripts/build.sh            # → bin/pixelpresent (linux/amd64, estático)
```

> La configuración concreta de infraestructura (Caddyfile, unit de systemd, script de despliegue) vive fuera del repositorio por decisión de proyecto.

---

## Estado

MVP desplegado y funcional en [pixelpresent.eu](https://pixelpresent.eu): magic link, editor, publicación, revelación por confeti y reacciones. Fuera del MVP (ver documentos de diseño): audio/foto, envío programado por email, más mecánicas de revelación, plantillas y regalo colaborativo.

---

## Licencia

[MIT](LICENSE) © 2026 Jairo Rivera
