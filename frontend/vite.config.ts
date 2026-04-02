import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import { execSync } from 'node:child_process'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

function safeGitSha() {
  try {
    return execSync('git rev-parse --short HEAD', { stdio: ['ignore', 'pipe', 'ignore'] }).toString().trim()
  } catch {
    return ''
  }
}

function readPackageVersion() {
  try {
    const pkgPath = resolve(process.cwd(), 'package.json')
    const raw = readFileSync(pkgPath, 'utf8')
    const pkg = JSON.parse(raw) as { version?: string }
    return pkg.version ?? ''
  } catch {
    return ''
  }
}

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const appVersion = env.VITE_APP_VERSION || env.APP_VERSION || process.env.npm_package_version || readPackageVersion()
  const repoUrl = env.VITE_REPO_URL || ''
  const buildTime = env.VITE_BUILD_TIME || new Date().toISOString()
  const gitSha = env.VITE_GIT_SHA || safeGitSha()

  return {
    plugins: [vue()],
    define: {
      __APP_VERSION__: JSON.stringify(appVersion),
      __BUILD_TIME__: JSON.stringify(buildTime),
      __GIT_SHA__: JSON.stringify(gitSha),
      __REPO_URL__: JSON.stringify(repoUrl),
    },
    server: {
      proxy: {
        '/v1': 'http://localhost:8080',
        '/r': 'http://localhost:8080',
        '/openapi.yaml': 'http://localhost:8080',
        '/docs': 'http://localhost:8080',
      },
    },
  }
})
