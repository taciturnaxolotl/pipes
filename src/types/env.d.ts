declare module "bun" {
  interface Env {
    ORIGIN: string;
    NODE_ENV?: "dev" | "production";
    PORT?: string;
  }
}
