> Code comments and this README were written by Claude (AI assistant)

# AutoTracking

A service for automatically marking student attendance in Google Sheets via QR code.

Each student carries a QR code linking to `http://<server>:8090/<course>?id=<studentID>`.
A camera at the entrance scans the code (`main.py`), which fires a GET request to that link — the
server (`main.go`) finds the right course's spreadsheet, finds the column for today's date and the
student's row, writes `1` into the cell, and colors it green.

## Project structure

```
main.go              — entry point: starts the logger, initializes Attendance and Course, starts the HTTP server
main.py              — separate script: scans QR codes via webcam and fires the link found in them
handler/handler.go   — HTTP handler: parses the URL (/<course>?id=<id>) and calls Attendance.Track
Attendance/          — Google Sheets API logic: finding the right cell by date/id, writing, coloring
Course/               — course -> spreadsheetID in memory, loaded from config.env
Conf/                 — config.env parser (COURSE=SpreadsheetID format)
logger/               — shared logger: writes to both console and Server.log
credentials.json      — Google service account (in .gitignore, not committed)
config.env            — list of courses and their spreadsheetIDs (in .gitignore)
config.env.example    — example config.env format
```

## Spreadsheet format

The sheet must be named `Attendance`. Expected structure:
- row 3 — dates (any format Google Sheets stores as a serial number)
- starting from row 4, column A — student ID, then attendance marks by date across the row

## Setup

1. Create a service account in Google Cloud, enable the Sheets API, download the JSON key and
   save it as `credentials.json` in the project root. Grant the service account edit access to
   the relevant spreadsheets.
2. Copy `config.env.example` to `config.env` and fill in your courses:
   ```
   MyCourse=1iDIl2sGg8nOBugNQfcSjmqDnq5lFuv2wL7wqQEKVTeU
   ```
   The value is the spreadsheet ID (from its Google Sheets URL).
3. `credentials.json` and `config.env` are already in `.gitignore` — secrets don't get committed.

## Running

```bash
go build -o autotracking .
./autotracking
```

The server starts on port `8090`. Logs are written to both the console and `Server.log`
(created/appended in the working directory).

## QR scanner (main.py)

A separate helper script in Python — scans QR codes via webcam and fires the link found in them
with a GET request, without opening a browser. Beeps on success/error, shows status fullscreen.

```bash
pip install opencv-python requests --break-system-packages
python3 main.py
```

`q` — quit, `f` — toggle fullscreen.

## Known quirks / things to watch out for

- A special `id=1488` in the handler marks a student as "absent" instead of "present" —
  currently hardcoded directly in the code. Should be moved to config.
- `go.mod` and `go.sum` are currently in `.gitignore` — usually these get committed for
  reproducible builds.