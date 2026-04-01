import { defineStore } from 'pinia'
import * as api from '../lib/api'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null as api.User | null,
    csrf: null as string | null,
    loading: false,
  }),
  getters: {
    isAuthed: (s) => !!s.user,
  },
  actions: {
    async bootstrap() {
      this.loading = true
      try {
        const me = await api.getMe()
        this.user = me
        const csrf = await api.getCsrf()
        this.csrf = csrf.token
      } catch {
        this.user = null
        this.csrf = null
      } finally {
        this.loading = false
      }
    },
    async login(email: string, password: string) {
      const resp = await api.login(email, password)
      this.user = resp.user
      const csrf = await api.getCsrf()
      this.csrf = csrf.token
    },
    async register(email: string, password: string) {
      await api.register(email, password)
      await this.login(email, password)
    },
    async logout() {
      if (this.csrf) {
        await api.logout(this.csrf)
      }
      this.user = null
      this.csrf = null
    },
  },
})

