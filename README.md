# Schul-App — Projektdokumentation

Eine webbasierte Schul-App mit Schüler- und Lehrer-Login, Vertretungsplan, Kalender, Dateiverwaltung, Chat und Hausaufgaben.

---

## Das System auf einen Blick

```
Browser → Reverse Proxy → Frontend (React) → Backend (Express) → Datenbank (PostgreSQL)
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
| Backend | Node.js + Express + TypeScript |
| Datenbank | PostgreSQL + Prisma ORM |
| Auth | JWT (Access Token im Memory + Refresh Token als HttpOnly-Cookie) |
| Echtzeit | Socket.io (Chat) |
| Datei-Upload | Multer |

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

Frontend und Backend nutzen einen gemeinsamen API-Vertrag (OpenAPI). Das hilft dabei, sich auf Endpunkte und Datenstrukturen zu einigen, Dokumentation nah am Code zu halten und Tippfehler bei Feldnamen früh zu erkennen.

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

**Socket.io Events:** `send_message` → `new_message`, `typing` → `user_typing`, `mark_read`

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
├── prisma/
│   ├── schema.prisma
│   └── seed.ts           1 Admin, 3 Lehrer, 10 Schüler, 2 Klassen
├── src/
│   ├── index.ts          Express-Server, Middleware
│   ├── config/           Prisma Client, Env-Validierung (zod)
│   ├── middleware/        JWT-Auth, Rollen-Check, Error-Handler
│   ├── routes/           Ein File pro Feature
│   └── socket/           Socket.io Handler (Chat)
└── uploads/              Lokaler Datei-Speicher
```

---

## Phasen-Plan

### Phase 1 — Auth & Fundament

**Ziel:** Lauffähige Infrastruktur, Login funktioniert end-to-end.

**Backend:**
1. Express + Prisma + PostgreSQL aufsetzen
2. Datenbankschema migrieren (`npx prisma migrate dev`)
3. `bcrypt`-Passwort-Hashing + JWT-Middleware
4. `POST /auth/login`, `POST /auth/refresh`, `GET /auth/me`
5. Rate Limiting auf Login-Endpunkt

**Frontend:**
1. Axios-Client mit Token-Interceptor (automatischer Refresh bei 401)
2. `authStore` mit Zustand: `user`, `accessToken`, `login()`, `logout()`
3. Access Token nur im Memory, Refresh Token als HttpOnly-Cookie
4. `LoginPage.tsx` mit Fehlerbehandlung
5. `ProtectedRoute` + `RoleRoute` (student / teacher / admin)
6. `AppLayout` + rollenbasierte `Sidebar`

**Sicherheit:**
- Passwörter: `bcrypt` mit `saltRounds: 12`
- Access Token: 15 Minuten Laufzeit, nur im Memory
- Refresh Token: 7 Tage, nur als HttpOnly-Cookie
- Rate Limiting: `express-rate-limit`

---

### Phase 2 — Benutzer & Klassen

**Ziel:** Admin kann Benutzer und Klassen verwalten.

1. CRUD-Endpunkte für Benutzer (Admin-only via `requireRole('admin')`)
2. CRUD-Endpunkte für Klassen + Mitglieder zuweisen
3. Seed-Skript ausführen
4. Admin-Seiten: Benutzertabelle mit Suche, Klassenverwaltung mit Zuweisung

---

### Phase 3 — Vertretungsplan

**Ziel:** Lehrer pflegen Vertretungen, Schüler sehen sie tagesaktuell.

1. CRUD-Endpunkte mit automatischer Filterung nach Klasse des Schülers
2. Tagesansicht mit Farbkodierung: Ausfall = rot, Vertretung = gelb, Raumänderung = blau
3. Auto-Refresh alle 5 Minuten (React Query `refetchInterval`)
4. Lehrer-Formular zum Erstellen/Bearbeiten

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

1. `multer` Upload (max 50 MB, MIME-Type-Prüfung)
2. Download nur mit gültigem JWT — kein direkter Dateizugriff
3. Ordner-Hierarchie mit rekursiver Query für Breadcrumb-Navigation
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

1. Socket.io mit JWT-Auth für Handshake-Authentifizierung
2. Socket-Events: `send_message` → broadcastet `new_message`, `typing`, `mark_read`
3. REST für Kanal-Erstellung + historische Nachrichten (Cursor-Pagination)
4. Zweispaltiges Layout: Kanalliste | Nachrichtenthread
5. Virtuelles Scrolling für alte Nachrichten, ungelesene Badges, Tipp-Indikator

---

### Phase 8 — Dashboard, Benachrichtigungen & Sicherheit

**Ziel:** Vollständig nutzbare, robuste App.

1. Rollenspezifisches Dashboard: Schüler sehen Vertretungen + Hausaufgaben + Termine, Lehrer sehen Klassen + Abgaben + Nachrichten
2. Benachrichtigungs-Tabelle + Socket.io Push + `NotificationBell`
3. Globale Error-Boundary, Toast-Notifications, Skeleton-Loader
4. Alle Inputs serverseitig mit `zod` validieren
5. CSRF-Schutz (`sameSite: 'strict'`), Dateinamen sanitisieren
6. Mobile Responsiveness: Sidebar als Drawer auf kleinen Screens

---

## Installation

### Backend

```bash
cd schulapp-backend
npm install express prisma @prisma/client bcrypt jsonwebtoken cookie-parser cors express-rate-limit zod multer socket.io
npm install -D typescript ts-node @types/express @types/bcrypt @types/jsonwebtoken @types/multer nodemon

# Datenbank migrieren
npx prisma migrate dev --name init

# Seed-Daten anlegen
npx ts-node prisma/seed.ts

# Server starten
npm run dev
```

### Frontend

```bash
cd codeclub-ui
npm install axios zustand @tanstack/react-query react-router-dom socket.io-client

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
Backend (Prozess)              →  Node-Server auf Port starten
Datenbank                      →  PostgreSQL-Dienst bereitstellen
```

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
| Backend/API | Express-Server | auf dem Server | Programm starten, als Prozess betreiben |
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
