# Schul-App — Projektdokumentation

Eine webbasierte Schul-App mit Schüler- und Lehrer-Login, Vertretungsplan, Kalender, Dateiverwaltung, Chat und Hausaufgaben.

---

## Das System auf einen Blick

```
Browser → Reverse Proxy → Frontend (React) → Backend (Go) → Datenbank (PostgreSQL)
```

| Teil | Aufgabe |
|---|---|
| **Browser** | Führt die React-App aus, ruft das Backend per `fetch` auf |
| **Frontend** | Zeigt Seiten und Komponenten, liest Formulareingaben, lädt Daten |
| **Backend** | Validiert Eingaben, setzt Regeln durch, stellt API-Endpunkte bereit |
| **Datenbank** | Speichert alle Daten dauerhaft (Nutzer, Klassen, Nachrichten, Dateien …) |
| **Reverse Proxy** | Nimmt HTTP-Requests an und leitet sie an Frontend oder Backend weiter |

---

## Tech-Stack

| Schicht | Technologie |
|---|---|
| Frontend | React + TypeScript + Vite + Tailwind CSS |
| Routing | React Router v7 |
| State | Zustand + React Query |
| Backend | Go + chi (HTTP-Router) |
| API-Vertrag | OpenAPI 3.1 + oapi-codegen |
| Datenbank | PostgreSQL + sqlc (typsichere SQL-Queries) |
| Migrationen | golang-migrate |
| Auth | JWT (Access Token im Memory + Refresh Token als HttpOnly-Cookie) |
| Echtzeit | gorilla/websocket (Chat) |
| Datei-Upload | Go stdlib `multipart` |

---

## Rendering-Ansatz: CSR

Diese App nutzt **Client Side Rendering**:

1. Browser lädt `index.html`, CSS und JavaScript aus `dist/`
2. React startet im Browser
3. React ruft per `fetch` das Backend auf
4. Seite aktualisiert sich mit den geladenen Daten

Das Frontend ist ein **statisches Deployment-Artefakt**: `npm run build` erzeugt `dist/` mit HTML, CSS und JavaScript — kein Server-Prozess nötig, nur ein Webserver der die Dateien ausliefert.

Das Backend ist ein **laufender Server-Prozess** auf einem Port. Dieser muss gestartet werden und auf seine Konfiguration (Datenbankverbindung, Secrets) zugreifen können.

---

## Wie Frontend und Backend kommunizieren

Der Browser ruft das Backend direkt per `fetch` auf:

```ts
const response = await fetch(`${apiBase}/api/v1/projects`);
const projects = await response.json();
```

Worauf geachtet werden muss:
- Die API-URL muss im Frontend bekannt sein (Umgebungsvariable)
- Der Browser braucht Zugriff auf die API (CORS konfigurieren)
- Secrets gehören nicht ins Frontend — sie bleiben auf dem Server

### OpenAPI als Vertrag

Die `openapi.yaml` ist die **einzige Quelle der Wahrheit** für alle Endpunkte, Request-Bodies und Response-Typen.

```
openapi.yaml
    ↓ oapi-codegen
schulapp-backend/internal/api/generated.go   ← Go Handler-Interfaces + Typen
    ↓ openapi-typescript-codegen (oder orval)
codeclub-ui/src/api/generated.ts             ← TypeScript-Typen + fetch-Wrapper
```

**Workflow:**
1. Endpunkt in `openapi.yaml` beschreiben
2. `make generate` läuft beide Codegeneratoren
3. Go: generierten Interface implementieren
4. TypeScript: generierten Client im Hook nutzen

Das verhindert Tippfehler bei Feldnamen, sorgt für konsistente Statuscodes und macht Breaking Changes sofort sichtbar — der Compiler meckert, bevor der Browser es tut.

**Vorteile gegenüber händisch geschriebenen Clients:**
- Kein manuelles Synchronisieren von Typen zwischen Go und TypeScript
- Automatische API-Dokumentation (z. B. mit Swagger UI oder Scalar)
- Neue Teammitglieder sehen sofort alle Endpunkte

**Wann es aufwendiger wird:**
- Die Spezifikation muss bei jedem Endpunkt-Umbau aktualisiert werden
- Generierter Code sieht manchmal ungewohnt aus — nicht anfassen, neu generieren

---

## Datenbankschema

### Kern-Tabellen

```sql
users           → id, email, password, role (student|teacher|admin), first_name, last_name
classes         → id, name, school_year
class_members   → class_id, user_id
class_teachers  → class_id, user_id, is_home_teacher
subjects        → id, name, short
refresh_tokens  → id, user_id, token, expires_at
```

### Feature-Tabellen

```sql
substitutions        → date, period, class_id, original_teacher_id, sub_teacher_id, room, type
events               → title, start_time, end_time, all_day, type, class_id (NULL = schulweit)
files                → name, path, size, mime_type, uploader_id, class_id, folder_id
file_folders         → name, parent_id (rekursiv), class_id
chat_channels        → type (direct|class|group), class_id
chat_members         → channel_id, user_id, last_read_at
messages             → channel_id, sender_id, content, file_id
homework             → title, due_date, class_id, subject_id, teacher_id
homework_submissions → homework_id, student_id, status (open|submitted|graded), grade
```

---

## API-Endpunkte

### Auth `/api/v1/auth`
```
POST  /login      E-Mail + Passwort → Access + Refresh Token
POST  /refresh    Refresh Token → neues Access Token
POST  /logout     Refresh Token invalidieren
GET   /me         Eigenes Profil
PATCH /me         Eigenes Profil aktualisieren
```

### Benutzer `/api/v1/users` (Admin)
```
GET    /          Alle Benutzer
POST   /          Benutzer erstellen
PATCH  /:id       Benutzer bearbeiten
DELETE /:id       Benutzer deaktivieren
```

### Klassen `/api/v1/classes`
```
GET    /                      Klassen (eigene für Schüler/Lehrer)
POST   /                      Klasse erstellen (Admin)
POST   /:id/members           Schüler hinzufügen
DELETE /:id/members/:uid      Schüler entfernen
POST   /:id/teachers          Lehrer zuweisen
```

### Vertretungsplan `/api/v1/substitutions`
```
GET    /     ?date=YYYY-MM-DD&classId=
POST   /     Eintrag erstellen (Lehrer/Admin)
PATCH  /:id  Bearbeiten
DELETE /:id  Löschen
```

### Kalender `/api/v1/events`
```
GET    /     ?from=&to=&classId=
POST   /     Termin erstellen (Lehrer)
PATCH  /:id  Bearbeiten (Ersteller/Admin)
DELETE /:id  Löschen
```

### Dateien `/api/v1/files`
```
GET    /folders         Ordnerstruktur
POST   /folders         Ordner erstellen
POST   /upload          Datei hochladen (multipart/form-data)
GET    /:id/download    Datei herunterladen (Auth erforderlich)
DELETE /:id             Datei löschen
```

### Chat `/api/v1/chat`
```
GET    /channels                        Eigene Kanäle
POST   /channels                        Kanal/DM erstellen
GET    /channels/:id/messages           Nachrichten (Cursor-Pagination)
POST   /channels/:id/messages           Nachricht senden
PATCH  /channels/:id/read               Gelesen-Markierung
```

**WebSocket Events:** `send_message` → `new_message`, `typing` → `user_typing`, `mark_read`

### Hausaufgaben `/api/v1/homework`
```
GET    /                   ?classId=&subjectId=&upcoming=true
POST   /                   Aufgabe erstellen (Lehrer)
GET    /:id                Details
PATCH  /:id                Bearbeiten
POST   /:id/submissions    Abgabe einreichen (Schüler)
GET    /:id/submissions    Alle Abgaben (Lehrer)
PATCH  /:id/submissions/:sid  Bewerten (Lehrer)
```

---

## Frontend-Struktur

```
codeclub-ui/src/
├── api/              Axios-Client + Endpunkte pro Feature
├── store/            Zustand Stores (Auth, Chat/Socket)
├── hooks/            React Query Hooks pro Feature
├── router/           Routen, ProtectedRoute, RoleRoute
├── layouts/          AuthLayout, AppLayout (mit Sidebar)
├── components/
│   ├── ui/           Button, Input, Modal, Badge, Avatar …
│   ├── layout/       Sidebar, Header, NotificationBell
│   └── shared/       FileUpload, DatePicker, RichTextEditor
└── pages/
    ├── auth/         LoginPage, ProfilePage
    ├── dashboard/    DashboardPage (rollenspezifisch)
    ├── substitutions/
    ├── calendar/
    ├── files/
    ├── chat/
    ├── homework/
    └── admin/        UsersPage, ClassesPage
```

---

## Backend-Struktur

```
schulapp-backend/
├── openapi.yaml                    API-Spezifikation (Quelle der Wahrheit)
├── Makefile                        make generate, make migrate, make run
├── cmd/
│   └── server/
│       └── main.go                 Entry Point, Dependency Wiring
├── internal/
│   ├── api/
│   │   ├── generated.go            oapi-codegen Output — nicht manuell bearbeiten
│   │   └── handler/                Implementierungen der generierten Interfaces
│   │       ├── auth.go
│   │       ├── users.go
│   │       ├── classes.go
│   │       ├── substitutions.go
│   │       ├── events.go
│   │       ├── files.go
│   │       ├── homework.go
│   │       └── chat.go
│   ├── db/
│   │   ├── query/                  SQL-Queries für sqlc
│   │   └── generated/              sqlc Output — nicht manuell bearbeiten
│   ├── middleware/
│   │   ├── auth.go                 JWT verifizieren, User in Context setzen
│   │   └── role.go                 Rollen prüfen
│   └── ws/
│       └── hub.go                  WebSocket Hub (Chat-Echtzeit)
├── migrations/                     SQL-Migrationsdateien (golang-migrate)
├── uploads/                        Lokaler Datei-Speicher
└── seed/
    └── main.go                     Testdaten: 1 Admin, 3 Lehrer, 10 Schüler
```

---

## Phasen-Plan

### Phase 1 — Auth & Fundament

**Ziel:** Lauffähige Infrastruktur, Login funktioniert end-to-end.

**Backend:**
1. Go-Modul anlegen, `chi`, `golang-jwt`, `bcrypt`, `sqlc`, `golang-migrate` hinzufügen
2. `openapi.yaml` mit Auth-Endpunkten beschreiben, `make generate` ausführen
3. Migrationen schreiben, `make migrate` ausführen
4. `bcrypt`-Passwort-Hashing + JWT-Middleware in `internal/middleware/auth.go`
5. `POST /auth/login`, `POST /auth/refresh`, `GET /auth/me` als Handler implementieren
6. Rate Limiting auf Login-Endpunkt (`golang.org/x/time/rate`)

**Frontend:**
1. Generierten TypeScript-Client aus `openapi.yaml` einbinden (`make generate`)
2. `authStore` mit Zustand: `user`, `accessToken`, `login()`, `logout()`
3. Access Token nur im Memory, Refresh Token als HttpOnly-Cookie
4. Axios-Interceptor für automatischen Token-Refresh bei 401
5. `LoginPage.tsx` mit Fehlerbehandlung
6. `ProtectedRoute` + `RoleRoute` (student / teacher / admin)
7. `AppLayout` + rollenbasierte `Sidebar`

**Sicherheit:**
- Passwörter: `bcrypt` mit Cost 12
- Access Token: 15 Minuten Laufzeit, nur im Memory
- Refresh Token: 7 Tage, nur als HttpOnly-Cookie
- Rate Limiting auf `/auth/login`

---

### Phase 2 — Benutzer & Klassen

**Ziel:** Admin kann Benutzer und Klassen verwalten.

1. Endpunkte in `openapi.yaml` ergänzen, `make generate`
2. CRUD-Handler für Benutzer (Middleware: `RequireRole("admin")`)
3. CRUD-Handler für Klassen + Mitglieder zuweisen
4. `go run seed/main.go` für Testdaten
5. Admin-Seiten: Benutzertabelle mit Suche, Klassenverwaltung mit Zuweisung

---

### Phase 3 — Vertretungsplan

**Ziel:** Lehrer pflegen Vertretungen, Schüler sehen sie tagesaktuell.

1. Endpunkte + Typen in `openapi.yaml`, `make generate`
2. Go-Handler mit automatischer Filterung nach Klasse des eingeloggten Schülers
3. Tagesansicht mit Farbkodierung: Ausfall = rot, Vertretung = gelb, Raumänderung = blau
4. Auto-Refresh alle 5 Minuten (React Query `refetchInterval`)
5. Lehrer-Formular zum Erstellen/Bearbeiten

---

### Phase 4 — Kalender

**Ziel:** Schulweite und klassenspezifische Termine.

1. Events-Endpunkte mit Datumsbereichs-Filterung
2. Sichtbarkeit: schulweit (class_id = NULL) vs. klassenspezifisch
3. Monats- und Wochenansicht mit `react-big-calendar`
4. Farbkodierung nach Typ: Ferien, Klassenarbeit, Veranstaltung
5. Termin-Erstellen-Modal für Lehrer

---

### Phase 5 — Dateiverwaltung

**Ziel:** Lehrer laden Materialien hoch, Schüler können herunterladen.

1. Go `multipart.Reader` für Upload (max 50 MB, MIME-Type-Prüfung mit `http.DetectContentType`)
2. Download-Handler prüft JWT — kein direkter Dateizugriff ohne Auth
3. Rekursive SQL-Query (CTE) für Ordner-Hierarchie + Breadcrumb
4. Explorer-Layout, Drag-and-Drop mit Fortschrittsanzeige, Dateivorschau

---

### Phase 6 — Hausaufgaben

**Ziel:** Lehrer erstellen Aufgaben, Schüler sehen Fälligkeiten und geben ab.

1. Beim Erstellen automatisch `submissions` für alle Klassenmitglieder anlegen (Status: `open`)
2. Status-Flow: `open` → `submitted` → `graded`
3. Schüler-Ansicht: Ampel-Status, gruppiert nach Fälligkeit, Filter: Alle / Offen / Diese Woche
4. Lehrer-Ansicht: Abgabeliste aller Schüler, Note eintragen, Rich-Text-Beschreibung

---

### Phase 7 — Chat

**Ziel:** Echtzeit-Kommunikation zwischen Lehrern und Schülern.

1. `gorilla/websocket` Hub in `internal/ws/hub.go` — ein goroutine pro Verbindung
2. JWT-Validierung beim WebSocket-Upgrade (Query-Parameter `?token=...`)
3. Nachrichten-Events: `send_message` → Hub broadcastet `new_message`, `typing`, `mark_read`
4. REST für Kanal-Erstellung + historische Nachrichten (Cursor-Pagination via sqlc)
5. Zweispaltiges Layout: Kanalliste | Nachrichtenthread
6. Virtuelles Scrolling, ungelesene Badges, Tipp-Indikator

---

### Phase 8 — Dashboard, Benachrichtigungen & Sicherheit

**Ziel:** Vollständig nutzbare, robuste App.

1. Rollenspezifisches Dashboard: Schüler sehen Vertretungen + Hausaufgaben + Termine, Lehrer sehen Klassen + Abgaben + Nachrichten
2. Benachrichtigungs-Tabelle + WebSocket Push + `NotificationBell`
3. Globale Error-Boundary, Toast-Notifications, Skeleton-Loader
4. Alle Inputs serverseitig in Go validieren (generierte Typen aus OpenAPI + eigene Checks)
5. CSRF-Schutz (`sameSite: 'strict'`), Dateinamen sanitisieren (`filepath.Base`)
6. Mobile Responsiveness: Sidebar als Drawer auf kleinen Screens

---

## Installation

### Voraussetzungen

- Go 1.22+
- Node.js 20+
- PostgreSQL (lokal oder Docker)
- `oapi-codegen`: `go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest`
- `sqlc`: `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`
- `golang-migrate`: `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`

### Backend

```bash
cd schulapp-backend

# Go-Abhängigkeiten
go mod tidy

# Code aus openapi.yaml und SQL-Queries generieren
make generate

# Datenbank migrieren
make migrate

# Seed-Daten anlegen
go run seed/main.go

# Server starten
go run cmd/server/main.go
# oder mit Live-Reload:
air
```

Umgebungsvariablen (`.env`):
```env
DATABASE_URL=postgres://user:pass@localhost:5432/schulapp
JWT_SECRET=your-secret-here
PORT=8080
UPLOAD_DIR=./uploads
```

### Frontend

```bash
cd codeclub-ui

# TypeScript-Client aus openapi.yaml generieren
npx @openapitools/openapi-generator-cli generate \
  -i ../schulapp-backend/openapi.yaml \
  -g typescript-axios -o src/api/generated

npm install axios zustand @tanstack/react-query react-router-dom

npm run dev       # Entwicklung
npm run build     # Produktion → dist/
```

---

## Backend — Mini-Checkliste

Startet das Backend immer zuerst ohne Frontend und testet es mit `curl`, Postman oder einer OpenAPI-UI:

- [ ] `GET /health` antwortet
- [ ] `GET /api/v1/items` liefert echte Daten
- [ ] `POST /api/v1/items` nimmt Daten an
- [ ] Datenbankanbindung funktioniert
- [ ] Fehlerfälle: ungültige Eingaben, nicht gefundene Daten, kaputte DB-Verbindung

Erst danach im Frontend anbinden.

---

## Deployment-Checkliste

```
Frontend (statische Dateien)   →  dist/ per Webserver ausliefern
Backend (Go Binary)            →  go build → einzelne Binary starten
Datenbank                      →  PostgreSQL-Dienst bereitstellen
```

Ein Go-Backend kompiliert zu einer **einzelnen statischen Binary** ohne externe Laufzeit. Das macht das Deployment einfacher als bei Node oder Python: Binary auf den Server kopieren, starten, fertig.

- [ ] Frontend gebaut (`npm run build`) und erreichbar?
- [ ] Backend läuft als Prozess auf dem Server?
- [ ] Frontend kennt die richtige Backend-URL (Umgebungsvariable)?
- [ ] Backend kann die Datenbank erreichen?
- [ ] Daten bleiben nach einem Neustart erhalten?
- [ ] Ansible-Playbook deployt reproduzierbar?
- [ ] GitHub Action ruft das Playbook auf?
- [ ] Zugangsdaten als GitHub Secrets hinterlegt — nicht im Repository?
- [ ] Kurze README erklärt das automatisierte Deployment?

---

## Statische Dateien vs. laufender Prozess

| Art | Beispiel | Läuft wo? | Deployment |
|---|---|---|---|
| Statische Dateien | Vite/React `dist/` | im Browser | Dateien kopieren und ausliefern |
| Backend/API | Go Binary | auf dem Server | `go build` → Binary kopieren und starten |
| Full-Stack-Framework | React Router v7, Next.js | Browser und Server | Server-Prozess + Assets |

---

## Ausblick: Was noch zu einem echten System gehört

Diese Themen sind nicht Teil des aktuellen Aufbaus, aber relevant wenn die App wächst:

| Thema | Kurz erklärt |
|---|---|
| **Reverse Proxy / Webserver** | Nimmt HTTP-Requests an und leitet sie an Frontend oder Backend weiter |
| **Bare Metal vs. Container** | Container isolieren Programme mit eigener Laufzeitumgebung (z. B. Docker) |
| **Horizontale / vertikale Skalierung** | Vertikal: ein Server wird stärker. Horizontal: mehrere Server teilen die Last |
| **Load Balancing** | Verteilt Requests auf mehrere laufende Instanzen |
| **File Storage / Buckets** | Für Uploads (Bilder, PDFs) statt Datenbankablage — z. B. S3-kompatibel |
| **Caching** | Häufig benötigte Daten zwischenspeichern, damit nicht jede Anfrage alles neu lädt |
| **Queues** | Aufgaben in Warteschlange legen und asynchron verarbeiten (z. B. E-Mails senden) |
| **Authentication / Authorization** | Auth prüft Identität, Authorization prüft Berechtigungen (z. B. Keycloak) |
| **Logging, Monitoring, Metrics** | Logging sammelt Fehler, Monitoring überwacht ob Systeme laufen, Metrics messen Laufzeiten |
