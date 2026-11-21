// Preload for the Node SSR process: exposes process.memoryUsage() on :13715/__mem
// so the Go harness can read RSS without calling `ps` (often blocked in sandboxes).
const http = require("node:http");

http
  .createServer((req, res) => {
    if (req.url === "/__mem") {
      const m = process.memoryUsage();
      res.setHeader("content-type", "application/json");
      res.end(
        JSON.stringify({
          rss: m.rss,
          heapTotal: m.heapTotal,
          heapUsed: m.heapUsed,
          external: m.external,
          pid: process.pid,
        }),
      );
      return;
    }
    res.statusCode = 404;
    res.end("not found");
  })
  .listen(13715, "127.0.0.1");
