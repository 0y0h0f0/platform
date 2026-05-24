import { configDefaults, mergeConfig, defineConfig } from 'vitest/config'

import viteConfig from './vite.config'

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      css: true,
      environment: 'jsdom',
      exclude: [...configDefaults.exclude, 'e2e/**', 'playwright-report/**', 'test-results/**'],
      globals: true,
      setupFiles: ['./__tests__/setup.ts'],
    },
  }),
)
