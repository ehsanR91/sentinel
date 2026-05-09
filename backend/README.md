
---

## Developer Quality-of-Life Mode

To make testing easier by removing production barriers (like having to set up the DB passwords or generate dummy keys), we have a `.dev` environment mode.

If you simply create a file named `.dev` inside the `backend/` directory:

```bash
touch backend/.dev
```

SentinelCore will automatically default to:
- Port: `8888`
- Default Secret Gate: `http://localhost:8888/dev/`
- Default Admin Username: `admin`
- Default Admin Password: `admin`
- Default Dummy `JWT_SECRET` injected automatically

Make sure to restart `air` if it is already running. The frontend will target `:8888` automatically via the updated vite dev proxy.
