# gRPC Server — Integration Test Cases

## Run

```bash
# unit tests only (no external deps)
go test ./face-recognition/grpc/server/... -v

# integration tests (requires CompreFace + MySQL)
go test -tags integration ./face-recognition/grpc/server/... -v
```

## Prerequisites

| Dependency | Purpose |
|---|---|
| CompreFace | Face recognition HTTP service called by `RecognizeImage` and `RecognizeBatch` |
| MySQL | Stores matched persons — queried by `GetSubjectFiles` |

Environment variables loaded from `.env`:

```
EXADEL_SERVICE_URL=http://localhost:8000/api/v1/recognition/recognize
EXADEL_API_KEY=your-api-key
TEST_DB_DSN=user:password@tcp(host:port)/dbname?parseTime=true
```

---

## Infrastructure

Integration tests use the same `bufconn` strategy as unit tests — gRPC runs in-process with no real port. The difference is the repo is backed by a **real MySQL connection** instead of a mock.

```
setupIntegrationClient(t)
  └─ godotenv.Load("../../../.env")
  └─ setupIntegrationDB(t)       → real MySQL
  └─ facerecognition.NewFaceRepo(db)
  └─ bufconn + grpc.NewServer    → in-process gRPC
  └─ returns (client, cleanup)
```

Call `defer cleanup()` in every test — it closes the connection, stops the server, and closes the DB.

---

## Test Cases

### TestGRPCRecognizeImage

**What it does:** Sends a single real image path through the gRPC `RecognizeImage` RPC. The server forwards it to CompreFace, parses the matched subjects, stores them in MySQL, and returns the persons list.

**Pipeline covered:**
```
gRPC client → RecognizeImage → GetFaceFromImage → CompreFace HTTP → InsertFaceAndImage → MySQL
```

**Assertions:**
- No error returned
- At least one person detected (`len(resp.Persons) > 0`)

**Output:**
```
RecognizeImage took 1.243s
Memory used: 0.12 MB
Persons detected: 1
  subject=phoebe  file=DSCF2733.JPG
```

---

### TestGRPCGetSubjectFiles

**What it does:** Queries all file names linked to subject `"phoebe"` from MySQL via the gRPC `GetSubjectFiles` RPC. No CompreFace call — pure DB read.

**Pipeline covered:**
```
gRPC client → GetSubjectFiles → GetFilesBySubject → MySQL SELECT
```

**Assertions:**
- No error returned
- `resp.Subject == "phoebe"`

**Output:**
```
GetSubjectFiles took 3ms
Subject: phoebe — 12 file(s)
  DSCF2733.JPG
  DSCF2778.JPG
  ...
```

**Note:** This test confirms that `GetSubjectFiles` is fast — the timing here reflects pure DB latency with no CompreFace involved. Use this as a baseline when comparing gRPC transport overhead.

---

### TestGRPCRecognizeBatch

**What it does:** Opens a bidirectional stream, sends 3 image paths one by one, and collects results as they arrive. The server processes each image via CompreFace concurrently with receiving the next request from the client.

**Pipeline covered:**
```
gRPC client (stream.Send × 3)
  → RecognizeBatch handler
    → GetFaceFromImage (per image)
    → CompreFace HTTP
    → InsertFaceAndImage → MySQL
    → stream.Send (result back)
gRPC client (stream.Recv loop)
```

**Assertions:**
- No send/recv errors
- Stream closes cleanly on EOF

**Output:**
```
  subject=phoebe  file=DSCF2733.JPG
  subject=vickie  file=DSCF2778.JPG
RecognizeBatch: 3 image(s), 2 person(s) total
Total time: 2.891s
Memory used: 0.31 MB
```

**Why this matters:** Compare `Total time` here against calling `TestGRPCRecognizeImage` three times sequentially. The streaming RPC returns results incrementally — the caller can act on the first result before the third image finishes processing.

---

## What is NOT covered (next steps)

| Scenario | Why it matters |
|---|---|
| `InsertFaceAndImage` DB failure mid-stream | Server logs the error and continues — verify stream stays open |
| `RecognizeBatch` with 50+ images | Validates memory stays stable under load |
| Subject not found in DB | `GetSubjectFiles` returns empty list, not an error — confirm behaviour |
| CompreFace returns 400 (unrecognised face) | Server maps it to `"nobody"` — verify this flows through gRPC correctly |
