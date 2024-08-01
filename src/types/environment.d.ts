declare global {
  namespace NodeJS {
    interface ProcessEnv {
      CURSEFORGE_API_KEY: string;
      MODRINTH_API_KEY: string;
      POSTHOG_API_KEY: string;
      npm_package_version: string;
    }
  }
}

export {};
