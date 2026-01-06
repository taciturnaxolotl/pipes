import { env } from "bun";
import indexHTML from "./pages/index.html";

(() => {
	const required = ["ORIGIN"];

	const missing = required.filter((key) => !process.env[key]);

	if (missing.length > 0) {
		console.warn(
			`[Startup] Missing required environment variables: ${missing.join(", ")}`,
		);
		process.exit(1);
	}

	// Validate ORIGIN is HTTPS in production
	const origin = process.env.ORIGIN as string;
	const nodeEnv = process.env.NODE_ENV || "development";

	if (nodeEnv === "production" && !origin.startsWith("https://")) {
		console.error(
			`[Startup] ORIGIN must use HTTPS in production (got: ${origin})`,
		);
		process.exit(1);
	}

	console.log(`[Startup] Environment validated (${nodeEnv} mode)`);
})();

const server = Bun.serve({
	port: env.PORT ? Number.parseInt(env.PORT, 10) : 3000,
	routes: {
		"/": indexHTML,
	},
	development: process.env.NODE_ENV !== "production",
});

console.log(`Pipes running on ${env.ORIGIN}`)

let is_shutting_down = false;
function shutdown(sig: string) {
	if (is_shutting_down) return;
	is_shutting_down = true;

	console.log(`[Shutdown] triggering shutdown due to ${sig}`);

	server.stop();
	console.log("[Shutdown] stopped server");

	process.exit(0);
}

process.on("SIGTERM", () => shutdown("SIGTERM"));
process.on("SIGINT", () => shutdown("SIGINT"));
